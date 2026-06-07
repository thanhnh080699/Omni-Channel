package adapterprocess

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"omni-channel/backend/internal/config"
)

type Process struct {
	cmd *exec.Cmd
}

func StartWhatsAppAdapter(ctx context.Context, cfg config.Config) (*Process, error) {
	if !cfg.WhatsAppAdapterAutostart {
		return nil, nil
	}
	if healthOK(ctx, cfg.WhatsAppAdapterURL) {
		log.Printf("whatsapp adapter already available at %s", cfg.WhatsAppAdapterURL)
		return nil, nil
	}

	dir, err := filepath.Abs(cfg.WhatsAppAdapterDir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		return nil, fmt.Errorf("whatsapp adapter package not found in %s: %w", dir, err)
	}
	if cfg.WhatsAppAdapterAutoInstall {
		if err := ensureDependencies(ctx, dir); err != nil {
			return nil, err
		}
	}

	args := cfg.WhatsAppAdapterArgs
	if len(args) == 0 {
		args = []string{"run", "dev"}
	}
	cmd := exec.Command(cfg.WhatsAppAdapterCommand, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"RABBITMQ_URL="+cfg.RabbitMQURL,
		"WHATSAPP_ADAPTER_PORT="+adapterPort(cfg.WhatsAppAdapterURL, "19090"),
		"WHATSAPP_SESSION_DIR="+filepath.Join(dir, "sessions"),
	)
	pipeOutput("whatsapp-adapter", cmd)

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	process := &Process{cmd: cmd}
	go func() {
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			log.Printf("whatsapp adapter exited: %v", err)
		}
	}()

	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for {
		if healthOK(waitCtx, cfg.WhatsAppAdapterURL) {
			log.Printf("whatsapp adapter started at %s", cfg.WhatsAppAdapterURL)
			return process, nil
		}
		if waitCtx.Err() != nil {
			process.Stop()
			return process, fmt.Errorf("whatsapp adapter did not become healthy at %s", cfg.WhatsAppAdapterURL)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func ensureDependencies(ctx context.Context, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); err == nil {
		return nil
	}
	log.Printf("installing whatsapp adapter dependencies in %s", dir)
	installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(installCtx, "npm", "install")
	cmd.Dir = dir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install whatsapp adapter dependencies: %w", err)
	}
	return nil
}

func (p *Process) Stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	if err := p.cmd.Process.Kill(); err != nil {
		log.Printf("stop whatsapp adapter: %v", err)
	}
}

func healthOK(ctx context.Context, baseURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func adapterPort(baseURL string, fallback string) string {
	parts := strings.Split(strings.TrimRight(baseURL, "/"), ":")
	if len(parts) == 0 {
		return fallback
	}
	port := parts[len(parts)-1]
	if port == "" || strings.Contains(port, "/") {
		return fallback
	}
	return port
}

func pipeOutput(prefix string, cmd *exec.Cmd) {
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		go copyLines(prefix, stdout)
	}
	stderr, err := cmd.StderrPipe()
	if err == nil {
		go copyLines(prefix, stderr)
	}
}

func copyLines(prefix string, reader io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			log.Printf("[%s] %s", prefix, strings.TrimSpace(string(buffer[:n])))
		}
		if err != nil {
			return
		}
	}
}
