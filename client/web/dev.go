// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package web

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// startDevServer starts the JS dev server that does on-demand rebuilding
// and serving of web client JS and CSS resources.
func (s *Server) startDevServer() (cleanup func()) {
	root := gitRootDir()
	webClientPath := filepath.Join(root, "client", "web")

	yarn := filepath.Join(root, "tool", "yarn")
	node := filepath.Join(root, "tool", "node")
	vite := filepath.Join(webClientPath, "node_modules", ".bin", "vite")

	log.Printf("installing JavaScript deps using %s... (might take ~30s)", yarn)
	out, err := exec.Command(yarn, "--non-interactive", "-s", "--cwd", webClientPath, "install").CombinedOutput()
	if err != nil {
		log.Fatalf("error running tailscale web's yarn install: %v, %s", err, out)
	}
	log.Printf("starting JavaScript dev server...")
	cmd := exec.Command(node, vite)
	cmd.Dir = webClientPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("Starting JS dev server: %v", err)
	}
	log.Printf("JavaScript dev server running as pid %d", cmd.Process.Pid)
	return func() {
		cmd.Process.Signal(os.Interrupt)
		err := cmd.Wait()
		log.Printf("JavaScript dev server exited: %v", err)
	}
}

func (s *Server) addProxyToDevServer() {
	if !s.devMode {
		return // only using Vite proxy in dev mode
	}
	// We use Vite to develop on the web client.
	// Vite starts up its own local server for development,
	// which we proxy requests to from Server.ServeHTTP.
	// Here we set up the proxy to Vite's server.
	handleErr := func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("The web client development server isn't running. " +
			"Run `./tool/yarn --cwd client/web start` from " +
			"the repo root to start the development server."))
		w.Write([]byte("\n\nError: " + err.Error()))
	}
	viteTarget, _ := url.Parse("http://127.0.0.1:4000")
	s.devProxy = httputil.NewSingleHostReverseProxy(viteTarget)
	s.devProxy.ErrorHandler = handleErr
}

func gitRootDir() string {
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		log.Fatalf("failed to find git top level (not in corp git?): %v", err)
	}
	return strings.TrimSpace(string(top))
}
