package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type serverControl struct {
    server   *http.Server
    running  bool
    mu       sync.Mutex
    cancelFn context.CancelFunc
}

func (sc *serverControl) startServer(apiURL, port string) {
    sc.mu.Lock()
    defer sc.mu.Unlock()

    if sc.running {
        return
    }

    _, cancel := context.WithCancel(context.Background())
    sc.cancelFn = cancel

    sc.server = &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: createHandler(apiURL)}
    go func() {
        sc.running = true
        log.Printf("Server is starting on :%s", port)
        if err := sc.server.ListenAndServe(); err != http.ErrServerClosed {
            log.Printf("Server error: %v", err)
        }
        sc.mu.Lock()
        sc.running = false
        sc.mu.Unlock()
    }()
}

func (sc *serverControl) stopServer() {
    sc.mu.Lock()
    defer sc.mu.Unlock()

    if !sc.running {
        return
    }

    log.Println("Stopping server...")
    sc.cancelFn()
    if err := sc.server.Shutdown(context.Background()); err != nil {
        log.Printf("Error shutting down server: %v", err)
    }
    sc.running = false
}

func createHandler(apiURL string) http.Handler {
    parsedAPIURL, err := url.Parse(apiURL)
    if err != nil {
        log.Fatalf("Invalid API URL: %v", err)
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "*")
        w.Header().Set("Access-Control-Allow-Headers", "*")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }

        targetURL := *parsedAPIURL
        targetURL.Path = r.URL.Path
        targetURL.RawQuery = r.URL.RawQuery

        req, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
        if err != nil {
            http.Error(w, "Failed to create request", http.StatusInternalServerError)
            return
        }

        for key, values := range r.Header {
            if strings.EqualFold(key, "Origin") {
                continue
            }
            for _, value := range values {
                req.Header.Add(key, value)
            }
        }

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            http.Error(w, "Failed to make request to target URL", http.StatusInternalServerError)
            return
        }
        defer resp.Body.Close()

        for key, values := range resp.Header {
            for _, value := range values {
                w.Header().Add(key, value)
            }
        }
        w.WriteHeader(resp.StatusCode)
        io.Copy(w, resp.Body)
    })
}

func main() {
    var apiURL string
    var port string

    // Parse command-line arguments
    flag.StringVar(&port, "p", "8764", "Port to run the server on")
    flag.StringVar(&port, "port", "8764", "Port to run the server on (alternative flag)")
    flag.Parse()

    if len(flag.Args()) > 0 {
        apiURL = flag.Arg(0)
    } else {
        log.Println("Warning: No API URL provided. You can enter it in the GUI.")
        apiURL = ""
    }

    sc := &serverControl{}

    go func() {
        w := new(app.Window)
        w.Option(app.Title("Un-CORS: Server Control"))
        w.Option(app.Size(unit.Dp(400), unit.Dp(300)))
        if err := draw(w, sc, apiURL, port); err != nil {
            log.Fatal(err)
        }
        os.Exit(0)
    }()
    app.Main()
}

func draw(w *app.Window, sc *serverControl, apiURL, port string) error {
    var ops op.Ops
    var startButton, stopButton widget.Clickable
    var apiInput, portInput widget.Editor
    th := material.NewTheme()

    // Default the API URL to "https://canvas.school.edu" if not provided
    if apiURL == "" {
        apiURL = "https://learn.school.edu"
    }

    // Prepopulate the text inputs with the command-line arguments or defaults
    apiInput.SetText(apiURL)
    portInput.SetText(port)

    // Define margins for the widgets
    margins := layout.Inset{
        Top:    unit.Dp(2),
        Bottom: unit.Dp(2),
        Left:   unit.Dp(10),
        Right:  unit.Dp(10),
    }

    for {
        e := w.Event()
        switch e := e.(type) {
        case app.DestroyEvent:
            return e.Err
        case app.FrameEvent:
            gtx := app.NewContext(&ops, e)

            layout.Flex{
                Axis:    layout.Vertical,
                Spacing: layout.SpaceEvenly,
            }.Layout(gtx,
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                    // Label for API URL
                    return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                        lbl := material.Label(th, unit.Sp(16), "Remote API URL")
                        return lbl.Layout(gtx)
                    })
                }),
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                    // Text input for API URL with margins, padding, and border
                    return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                        border := widget.Border{
                            Color:        th.Palette.ContrastBg,
                            CornerRadius: unit.Dp(4),
                            Width:        unit.Dp(2),
                        }
                        return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                            padding := layout.Inset{
                                Top:    unit.Dp(8),
                                Bottom: unit.Dp(8),
                                Left:   unit.Dp(12),
                                Right:  unit.Dp(12),
                            }
                            return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                                ed := material.Editor(th, &apiInput, "Enter API URL")
                                apiInput.SingleLine = true
                                return ed.Layout(gtx)
                            })
                        })
                    })
                }),
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                    // Label for Port
                    return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                        lbl := material.Label(th, unit.Sp(16), "Local Listening Port")
                        return lbl.Layout(gtx)
                    })
                }),
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                    // Text input for Port with margins, padding, and border
                    return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                        border := widget.Border{
                            Color:        th.Palette.ContrastBg,
                            CornerRadius: unit.Dp(4),
                            Width:        unit.Dp(2),
                        }
                        return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                            padding := layout.Inset{
                                Top:    unit.Dp(8),
                                Bottom: unit.Dp(8),
                                Left:   unit.Dp(12),
                                Right:  unit.Dp(12),
                            }
                            return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                                ed := material.Editor(th, &portInput, "Enter Port")
                                portInput.SingleLine = true
                                return ed.Layout(gtx)
                            })
                        })
                    })
                }),
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                    // Start button with margins
                    return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                        btn := material.Button(th, &startButton, "Start Server")
                        if startButton.Clicked(gtx) {
                            // Use the text from the inputs as the API URL and port
                            sc.startServer(apiInput.Text(), portInput.Text())
                        }
                        return btn.Layout(gtx)
                    })
                }),
                layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                    // Stop button with margins
                    return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                        btn := material.Button(th, &stopButton, "Stop Server")
                        if stopButton.Clicked(gtx) {
                            sc.stopServer()
                        }
                        return btn.Layout(gtx)
                    })
                }),
            )
            e.Frame(gtx.Ops)
        }
    }
}