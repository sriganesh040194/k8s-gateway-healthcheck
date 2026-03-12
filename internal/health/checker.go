package health

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Status represents the health state
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Response is the standard health response envelope
type Response struct {
	Status    Status           `json:"status"`
	Service   string           `json:"service"`
	Version   string           `json:"version"`
	Env       string           `json:"environment"`
	Timestamp string           `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	Checks    map[string]Check `json:"checks,omitempty"`
}

// Check is a single named health check result
type Check struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// HTTPStatus maps health status to HTTP status codes
func (r *Response) HTTPStatus() int {
	switch r.Status {
	case StatusHealthy, StatusDegraded:
		return http.StatusOK
	default:
		return http.StatusServiceUnavailable
	}
}

// Checker holds app metadata and startup state
type Checker struct {
	appName     string
	version     string
	env         string
	startTime   time.Time
	startupDone atomic.Bool
}

// NewChecker creates a Checker and marks startup complete after 1s
func NewChecker(appName, version, env string) *Checker {
	c := &Checker{
		appName:   appName,
		version:   version,
		env:       env,
		startTime: time.Now(),
	}
	go func() {
		time.Sleep(1 * time.Second)
		c.startupDone.Store(true)
	}()
	return c
}

// Liveness — is the process alive and not deadlocked?
// Kubernetes uses this to decide if the container needs a restart.
func (c *Checker) Liveness() *Response {
	return &Response{
		Status:    StatusHealthy,
		Service:   c.appName,
		Version:   c.version,
		Env:       c.env,
		Timestamp: now(),
		Uptime:    uptime(c.startTime),
	}
}

// Readiness — is the app ready to accept traffic?
// Kubernetes uses this to decide if the pod should receive traffic.
func (c *Checker) Readiness() *Response {
	checks := map[string]Check{}
	status := StatusHealthy

	if !c.startupDone.Load() {
		status = StatusUnhealthy
		checks["startup"] = Check{Status: StatusUnhealthy, Message: "Still initializing"}
	} else {
		checks["startup"] = Check{Status: StatusHealthy, Message: "Initialization complete"}
	}

	return &Response{
		Status:    status,
		Service:   c.appName,
		Version:   c.version,
		Env:       c.env,
		Timestamp: now(),
		Uptime:    uptime(c.startTime),
		Checks:    checks,
	}
}

// Startup — has the app completed initialization?
// Kubernetes uses this before switching to liveness/readiness probes.
func (c *Checker) Startup() *Response {
	if !c.startupDone.Load() {
		return &Response{
			Status:    StatusUnhealthy,
			Service:   c.appName,
			Version:   c.version,
			Env:       c.env,
			Timestamp: now(),
			Uptime:    uptime(c.startTime),
		}
	}
	return &Response{
		Status:    StatusHealthy,
		Service:   c.appName,
		Version:   c.version,
		Env:       c.env,
		Timestamp: now(),
		Uptime:    uptime(c.startTime),
	}
}

// Full — detailed status for App Gateway / monitoring dashboards
func (c *Checker) Full() *Response {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	heapMB := memStats.HeapAlloc / 1024 / 1024
	goroutines := runtime.NumGoroutine()

	checks := map[string]Check{
		"liveness": {Status: StatusHealthy, Message: "Process alive"},
	}

	// Memory check — warn if heap > 400MB
	memStatus := StatusHealthy
	memMsg := heapMsg(heapMB)
	if heapMB > 400 {
		memStatus = StatusDegraded
	}
	checks["memory"] = Check{Status: memStatus, Message: memMsg}

	// Goroutine leak check — warn if > 500
	goroutineStatus := StatusHealthy
	goroutineMsg := goroutineMsg(goroutines)
	if goroutines > 500 {
		goroutineStatus = StatusDegraded
	}
	checks["goroutines"] = Check{Status: goroutineStatus, Message: goroutineMsg}

	// Startup
	overallStatus := StatusHealthy
	if !c.startupDone.Load() {
		overallStatus = StatusUnhealthy
		checks["startup"] = Check{Status: StatusUnhealthy, Message: "Still initializing"}
	} else {
		checks["startup"] = Check{Status: StatusHealthy, Message: "Initialization complete"}
	}

	// Degrade overall if any check is degraded
	if overallStatus == StatusHealthy {
		for _, ch := range checks {
			if ch.Status == StatusDegraded {
				overallStatus = StatusDegraded
				break
			}
		}
	}

	return &Response{
		Status:    overallStatus,
		Service:   c.appName,
		Version:   c.version,
		Env:       c.env,
		Timestamp: now(),
		Uptime:    uptime(c.startTime),
		Checks:    checks,
	}
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func uptime(start time.Time) string {
	return time.Since(start).Round(time.Second).String()
}

func heapMsg(mb uint64) string {
	return fmt.Sprintf("%dMB heap allocated", mb)
}

func goroutineMsg(n int) string {
	return fmt.Sprintf("%d goroutines running", n)
}
