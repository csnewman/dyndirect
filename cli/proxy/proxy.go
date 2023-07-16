package proxy

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/csnewman/dyndirect/cli"
	dsdm "github.com/csnewman/dyndirect/go"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
)

//nolint:gochecknoglobals
var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	dotStyle     = helpStyle.Copy().UnsetMargins()
	appStyle     = lipgloss.NewStyle().Margin(1, 2, 0, 2)
)

type state struct {
	errored bool
	domain  string
	msg     string
	ready   bool
}

type requestMsg struct {
	when    time.Time
	remote  string
	request string
}

func (r requestMsg) String() string {
	if r.remote == "" {
		return dotStyle.Render(strings.Repeat(".", 60))
	}

	return fmt.Sprintf("[%s] %s: %s", r.when.Format("2006-01-02 15:04:05"), r.remote, r.request)
}

type model struct {
	state    state
	spinner  spinner.Model
	results  []requestMsg
	quitting bool
	src      int
	dst      string
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true

			return m, tea.Quit
		}

		return m, nil
	case state:
		m.state = msg

		if msg.errored {
			m.quitting = true

			return m, tea.Quit
		}

		return m, nil
	case requestMsg:
		m.results = append(m.results[1:], msg)

		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.quitting {
		return appStyle.Render("Proxy stopped\n\n")
	}

	var s string

	if m.state.ready {
		s += fmt.Sprintf("Active: https://127-0-0-1-v4.%s:%d -> %s", m.state.domain, m.src, m.dst)
		s += fmt.Sprintf("\nYou can also visit http://127.0.0.1:%d to be redirected to HTTPS.", m.src)
		s += "\n\n"

		for _, res := range m.results {
			s += res.String() + "\n"
		}
	} else {
		s += m.spinner.View() + " " + m.state.msg + "\n"
	}

	if m.quitting {
		s += "\n"
	} else {
		s += helpStyle.Render("Press q to exit")
	}

	return appStyle.Render(s)
}

func RunProxy(src int, dst string, overwriteHost bool, replace bool) {
	s := spinner.New()
	s.Style = spinnerStyle

	p := tea.NewProgram(model{
		spinner: s,
		results: make([]requestMsg, 10),
		src:     src,
		dst:     dst,
		state: state{
			domain: "",
			msg:    "Checking",
			ready:  false,
		},
	})

	outerCtx, cancel := context.WithCancel(context.Background())
	egrp, ctx := errgroup.WithContext(outerCtx)

	defer cancel()

	egrp.Go(func() error {
		err := startProxy(ctx, p, src, dst, overwriteHost, replace)
		if err != nil {
			p.Send(state{
				errored: true,
			})
		}

		return err
	})

	egrp.Go(func() error {
		_, err := p.Run()

		cancel()

		return err
	})

	if err := egrp.Wait(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return
		}

		fmt.Println("Error running program:", err) //nolint:forbidigo
		os.Exit(1)                                 //nolint:gocritic
	}
}

func startProxy(ctx context.Context, p *tea.Program, src int, dst string, overwriteHost bool, replace bool) error {
	var (
		domain string
		err    error
	)

	if !replace {
		domain, err = cli.GetDomain()
		if err != nil {
			return err
		}
	}

	newState := state{}

	if domain == "" {
		newState.msg = "Acquiring domain"
		p.Send(newState)

		domain, err = cli.IssueDomain(ctx)
		if err != nil {
			return err
		}
	}

	newState.domain = domain

	hasCert, err := cli.HasCertificate()
	if err != nil {
		return err
	}

	if !hasCert {
		newState.msg = "Acquiring certificate"
		p.Send(newState)

		if err := cli.AcquireCertificate(ctx); err != nil {
			return err
		}
	}

	cert, err := cli.GetCertificate()
	if err != nil {
		return err
	}

	newState.msg = "Ready"
	newState.ready = true
	p.Send(newState)

	target, err := url.Parse(dst)
	if err != nil {
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorLog = log.New(io.Discard, "", 0)

	if overwriteHost {
		oldDirector := proxy.Director
		proxy.Director = func(r *http.Request) {
			oldDirector(r)
			r.Host = r.URL.Host
		}
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", src))
	if err != nil {
		return err
	}

	// Create a mux.
	m := cmux.New(l)
	httpl := m.Match(cmux.HTTP1Fast())
	tlsl := m.Match(cmux.Any())

	go serveHTTP1(httpl, src, domain)
	go serveHTTPS(tlsl, cert, p, proxy)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- m.Serve()
	}()
	select {
	case <-ctx.Done():
		m.Close()

		return nil
	case err := <-serverErr:
		return err
	}
}

func serveHTTP1(l net.Listener, src int, domain string) {
	s := &http.Server{ //nolint:gosec
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, port, err := net.SplitHostPort(r.Host)
			if err != nil {
				port = strconv.Itoa(src)
			} else {
				if ip := net.ParseIP(host); ip != nil {
					host = dsdm.GetDomainForIP(domain, ip)
				}
			}

			u := r.URL
			u.Host = net.JoinHostPort(host, port)
			u.Scheme = "https"

			http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
		}),
	}

	if err := s.Serve(l); !errors.Is(err, cmux.ErrListenerClosed) {
		panic(err)
	}
}

func serveHTTPS(l net.Listener, cert tls.Certificate, p *tea.Program, proxy *httputil.ReverseProxy) {
	config := &tls.Config{ //nolint:gosec
		Certificates: []tls.Certificate{cert},
		Rand:         rand.Reader,
	}

	// Create TLS listener.
	tlsl := tls.NewListener(l, config)

	s := &http.Server{ //nolint:gosec
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			msg := requestMsg{
				when:    time.Now(),
				remote:  r.RemoteAddr,
				request: r.RequestURI,
			}

			go func() {
				p.Send(msg)
			}()

			proxy.ServeHTTP(w, r)
		}),
	}

	if err := s.Serve(tlsl); !errors.Is(err, cmux.ErrListenerClosed) {
		panic(err)
	}
}
