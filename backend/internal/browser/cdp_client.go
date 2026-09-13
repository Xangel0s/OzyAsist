package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type CDPClient struct {
	remoteURL string
	tempDir   string
}

type BrowserAction struct {
	Action   string `json:"action"` // 'navigate', 'click', 'type', 'screenshot', 'extract_text', 'eval_js'
	URL      string `json:"url,omitempty"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
	Script   string `json:"script,omitempty"`
}

type ActionResult struct {
	Success    bool   `json:"success"`
	Data       string `json:"data,omitempty"`
	Screenshot []byte `json:"-"`
	Error      string `json:"error,omitempty"`
}

func NewCDPClient(remoteDebuggingURL string) *CDPClient {
	return &CDPClient{
		remoteURL: remoteDebuggingURL,
		tempDir:   filepath.Join(os.TempDir(), "ozy_cdp_profile"),
	}
}

// CheckRemotePort verifica si hay un navegador Chrome escuchando en el puerto 9222
func (c *CDPClient) CheckRemotePort(ctx context.Context) bool {
	if c.remoteURL == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.remoteURL+"/json/version", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// createExecContext inicializa el contexto Chromedp (remoto si está activo, o headless nuevo)
func (c *CDPClient) createExecContext(parent context.Context) (context.Context, context.CancelFunc, error) {
	if c.CheckRemotePort(parent) {
		// Reutiliza la sesión remota abierta en el puerto 9222
		allocCtx, allocCancel := chromedp.NewRemoteAllocator(parent, c.remoteURL)
		taskCtx, taskCancel := chromedp.NewContext(allocCtx)
		cancel := func() {
			taskCancel()
			allocCancel()
		}
		return taskCtx, cancel, nil
	}

	// Lanza proceso Chromium 100% Headless e invisible
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("user-data-dir", c.tempDir),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(parent, opts...)
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	cancel := func() {
		taskCancel()
		allocCancel()
	}
	return taskCtx, cancel, nil
}

// ExecuteAction procesa comandos web en segundo plano
func (c *CDPClient) ExecuteAction(ctx context.Context, rawPayload string) (*ActionResult, error) {
	var action BrowserAction
	if err := json.Unmarshal([]byte(rawPayload), &action); err != nil {
		return nil, fmt.Errorf("error deserializando payload CDP: %w", err)
	}

	execCtx, cancel, err := c.createExecContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creando contexto de navegador: %w", err)
	}
	defer cancel()

	// Timeout de seguridad para la operación web
	timeoutCtx, timeoutCancel := context.WithTimeout(execCtx, 30*time.Second)
	defer timeoutCancel()

	result := &ActionResult{Success: true}

	switch action.Action {
	case "navigate":
		if action.URL == "" {
			return nil, errors.New("URL no especificada para acción navigate")
		}
		err = chromedp.Run(timeoutCtx, chromedp.Navigate(action.URL))
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Data = fmt.Sprintf("Navegación exitosa hacia %s", action.URL)
		}

	case "extract_text":
		var text string
		if action.Selector == "" {
			action.Selector = "body"
		}
		err = chromedp.Run(timeoutCtx,
			chromedp.Text(action.Selector, &text, chromedp.ByQuery),
		)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Data = text
		}

	case "type":
		err = chromedp.Run(timeoutCtx,
			chromedp.SendKeys(action.Selector, action.Value, chromedp.ByQuery),
		)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Data = fmt.Sprintf("Texto ingresado en %s", action.Selector)
		}

	case "click":
		err = chromedp.Run(timeoutCtx,
			chromedp.Click(action.Selector, chromedp.ByQuery),
		)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Data = fmt.Sprintf("Clic ejecutado en %s", action.Selector)
		}

	case "screenshot":
		var buf []byte
		err = chromedp.Run(timeoutCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				buf, err = page.CaptureScreenshot().WithFormat(page.CaptureScreenshotFormatPng).Do(ctx)
				return err
			}),
		)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Screenshot = buf
			result.Data = fmt.Sprintf("Captura capturada (%d bytes)", len(buf))
		}

	case "eval_js":
		var res interface{}
		err = chromedp.Run(timeoutCtx,
			chromedp.Evaluate(action.Script, &res),
		)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			resBytes, _ := json.Marshal(res)
			result.Data = string(resBytes)
		}

	default:
		return nil, fmt.Errorf("acción CDP no soportada: %s", action.Action)
	}

	return result, nil
}
