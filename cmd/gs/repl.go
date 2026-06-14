package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/runtime"
)

const replHelp = `.help        Show this help
.exit        Exit the REPL
.load <file> Load and evaluate a GoScript file`

func (r *runner) runREPL(cfg replConfig) error {
	if cfg.in == nil {
		cfg.in = os.Stdin
	}
	if cfg.out == nil {
		cfg.out = os.Stdout
	}
	if cfg.errOut == nil {
		cfg.errOut = os.Stderr
	}
	session, err := r.newREPLSession()
	if err != nil {
		return err
	}
	defer session.close()

	if cfg.showIntro {
		fmt.Fprintln(cfg.out, "GoScript "+version)
		fmt.Fprintln(cfg.out, "Type .help for commands, .exit to quit.")
	}

	scanner := bufio.NewScanner(cfg.in)
	var pending []string
	for {
		if len(pending) == 0 {
			fmt.Fprint(cfg.out, "gs> ")
		} else {
			fmt.Fprint(cfg.out, "... ")
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if len(pending) == 0 && strings.HasPrefix(strings.TrimSpace(line), ".") {
			if done := session.handleCommand(strings.TrimSpace(line), cfg.out, cfg.errOut); done {
				return nil
			}
			continue
		}
		pending = append(pending, line)
		src := strings.Join(pending, "\n")
		if replNeedsMoreInput(src) {
			continue
		}
		session.evalAndPrint(src, cfg.out, cfg.errOut)
		pending = nil
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(pending) > 0 {
		session.evalAndPrint(strings.Join(pending, "\n"), cfg.out, cfg.errOut)
	}
	return nil
}

type replSession struct {
	r   *runner
	sess *runtime.Session
	env  *object.Environment
	cwd  string
}

func (r *runner) newREPLSession() (*replSession, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	// A REPL keeps one long-lived Session so that bindings, required modules,
	// and async state persist across input lines. The plugin host (if any on a
	// future REPL-with-plugins path) would wire in here via NativeResolver.
	r.sess = runtime.NewSession(runtime.Options{
		Workers: r.opts.workers,
		Timeout: 0, // REPL never auto-times out; the user controls the session.
		CheckTypes: r.opts.checkTypes,
		RootDir: module.FindProjectRoot(cwd),
	})
	r.sess.VM().SetArgv([]string{executableArgv0(), "<repl>"})
	env := r.sess.NewEnvironment()
	r.sess.Configure(env, cwd)
	return &replSession{r: r, sess: r.sess, env: env, cwd: cwd}, nil
}

func (s *replSession) close() {
	if s.sess == nil {
		return
	}
	_ = s.sess.Drain()
	s.sess.Close()
	s.sess = nil
	s.r.sess = nil
}

func (s *replSession) handleCommand(line string, out, errOut io.Writer) bool {
	switch {
	case line == ".exit" || line == ".quit":
		return true
	case line == ".help":
		fmt.Fprintln(out, replHelp)
	case line == ".load" || strings.HasPrefix(line, ".load "):
		path := strings.TrimSpace(strings.TrimPrefix(line, ".load"))
		if path == "" {
			fmt.Fprintln(errOut, ".load requires a file path")
			return false
		}
		s.load(path, out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command: %s\n", line)
	}
	return false
}

func (s *replSession) load(path string, out, errOut io.Writer) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.cwd, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return
	}
	s.evalAndPrint(string(data), out, errOut)
}

func (s *replSession) evalAndPrint(src string, out, errOut io.Writer) {
	result, err := s.sess.EvalSource(src, "<repl>", s.env)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return
	}
	if err := s.sess.Drain(); err != nil {
		fmt.Fprintln(errOut, err)
		return
	}
	if result != nil && result != object.UNDEFINED {
		fmt.Fprintln(out, result.Inspect())
	}
}

func replNeedsMoreInput(src string) bool {
	state, err := scanREPLInput(src)
	if err != nil {
		return false
	}
	return state.depth > 0 || state.inString != 0 || state.inBlockComment
}

type replInputState struct {
	depth          int
	inString       rune
	inLineComment  bool
	inBlockComment bool
	escape         bool
}

func scanREPLInput(src string) (replInputState, error) {
	state := replInputState{}
	for _, ch := range src {
		if state.inLineComment {
			if ch == '\n' || ch == '\r' {
				state.inLineComment = false
			}
			continue
		}
		if state.inBlockComment {
			if state.escape && ch == '/' {
				state.inBlockComment = false
				state.escape = false
				continue
			}
			state.escape = ch == '*'
			continue
		}
		if state.inString != 0 {
			if state.escape {
				state.escape = false
				continue
			}
			if ch == '\\' {
				state.escape = true
				continue
			}
			if ch == state.inString {
				state.inString = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '`':
			state.inString = ch
		case '/':
			if state.escape {
				state.inLineComment = true
				state.escape = false
			} else {
				state.escape = true
			}
		case '*':
			if state.escape {
				state.inBlockComment = true
				state.escape = false
			}
		case '{', '(', '[':
			state.depth++
			state.escape = false
		case '}', ')', ']':
			if state.depth > 0 {
				state.depth--
			} else {
				return state, errors.New("unmatched closing delimiter")
			}
			state.escape = false
		default:
			state.escape = false
		}
	}
	return state, nil
}
