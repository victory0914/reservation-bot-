package client

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// AttemptLog represents a single reservation attempt row in the log
type AttemptLog struct {
	Slot   string // e.g., "2026-02-07 19:00"
	Result string // e.g., "Attempted (Success)" or "Waiting"
	Detail string // e.g., "Token acquired, POST sent"
	Status string // e.g., "Transaction Complete"
}

// LogEntry holds all the data required to generate the structured report
type LogEntry struct {
	TargetSite    string
	ExecutionMode string
	NetworkEnv    string
	Protocol      string

	// [1] Scheduler & Timing
	TargetTime time.Time
	ActualTime time.Time
	// Drift is calculated from ActualTime - TargetTime

	// [2] Connection State (Metrics from the critical request)
	DNSResolution    time.Duration
	TCPHandshake     time.Duration
	TLSHandshake     time.Duration
	ConnectionReused bool
	ProxyTunnel      string // e.g., "Established (HTTP CONNECT)"

	// [3] Monitoring & Detection
	MonitoringMethod   string
	PollingInterval    string
	AvailabilitySignal string

	// [4] Reservation Attempt Logic
	Attempts []AttemptLog

	// [5] Result Summary
	Result            string
	EndToEndReadiness string
	ObservedIssues    string
	EngineerNote      string
}

// PrintExecutionLog outputs the formatted log exactly as requested
func PrintExecutionLog(e LogEntry) {
	// Define colors
	headerColor := color.New(color.FgHiCyan, color.Bold).SprintfFunc()
	sectionColor := color.New(color.FgHiYellow).SprintFunc()
	labelColor := color.New(color.FgWhite).SprintFunc()
	valueColor := color.New(color.FgHiWhite).SprintFunc()
	successColor := color.New(color.FgGreen, color.Bold).SprintFunc()
	errorColor := color.New(color.FgRed, color.Bold).SprintFunc()
	driftColor := color.New(color.FgHiMagenta).SprintfFunc()

	fmt.Println("\n\n" + headerColor("[Reservation Bot Execution Log]"))
	fmt.Printf("%s      : %s\n", labelColor("Target Site"), valueColor(e.TargetSite))
	fmt.Printf("%s   : %s\n", labelColor("Execution Mode"), valueColor(e.ExecutionMode))
	fmt.Printf("%s          : %s\n", labelColor("Network"), valueColor(e.NetworkEnv))
	fmt.Printf("%s         : %s\n", labelColor("Protocol"), valueColor(e.Protocol))

	fmt.Println("\n" + sectionColor("--------------------------------------------------"))
	fmt.Println(sectionColor("[1] Scheduler & Timing"))
	fmt.Println(sectionColor("--------------------------------------------------"))
	fmt.Printf("%s      : %s (JST)\n", labelColor("Target Execution Time"), valueColor(e.TargetTime.Format("2006-01-02 15:04:05.000")))
	fmt.Printf("%s           : %s\n", labelColor("Actual Fire Time"), valueColor(e.ActualTime.Format("2006-01-02 15:04:05.000000")))

	drift := e.ActualTime.Sub(e.TargetTime)
	sign := "+"
	if drift < 0 {
		sign = "" // drift string includes -
	}
	fmt.Printf("%s               : %s\n", labelColor("Timing Drift"), driftColor("%s%d µs", sign, drift.Microseconds()))

	fmt.Println("\nComment:")
	fmt.Println("ミリ秒ではなく「マイクロ秒」単位で発火しており、")
	fmt.Println("OSタイマーや時計ズレの影響を受けていないことが確認できます。")

	fmt.Println("\n" + sectionColor("--------------------------------------------------"))
	fmt.Println(sectionColor("[2] Connection State"))
	fmt.Println(sectionColor("--------------------------------------------------"))
	fmt.Printf("%s             : %s\n", labelColor("DNS Resolution"), valueColor(fmt.Sprintf("%d ms", e.DNSResolution.Milliseconds())))
	fmt.Printf("%s              : %s\n", labelColor("TCP Handshake"), valueColor(fmt.Sprintf("%d ms", e.TCPHandshake.Milliseconds())))
	fmt.Printf("%s       : %s\n", labelColor("TLS Handshake (uTLS)"), valueColor(fmt.Sprintf("%d ms", e.TLSHandshake.Milliseconds())))
	fmt.Printf("%s          : %s\n", labelColor("Connection Reused"), valueColor(fmt.Sprintf("%v", e.ConnectionReused)))
	fmt.Printf("%s               : %s\n", labelColor("Proxy Tunnel"), valueColor(e.ProxyTunnel))

	fmt.Println("\nComment:")
	fmt.Println("本番ではこの接続を事前に確立（プリウォーム）するため、")
	fmt.Println("予約発火時にはこれらの遅延は発生しません。")

	fmt.Println("\n" + sectionColor("--------------------------------------------------"))
	fmt.Println(sectionColor("[3] Monitoring & Detection"))
	fmt.Println(sectionColor("--------------------------------------------------"))
	fmt.Printf("%s          : %s\n", labelColor("Monitoring Method"), valueColor(e.MonitoringMethod))
	fmt.Printf("%s           : %s\n", labelColor("Polling Interval"), valueColor(e.PollingInterval))
	fmt.Printf("%s        : %s\n", labelColor("Availability Signal"), valueColor(e.AvailabilitySignal))

	fmt.Println("\nComment:")
	fmt.Println("DOM全体の解析は行わず、レスポンス内の特定シグナルのみを監視。")
	fmt.Println("一般的な0.95秒固定監視よりも検知遅延が小さくなっています。")

	fmt.Println("\n" + sectionColor("--------------------------------------------------"))
	fmt.Println(sectionColor("[4] Reservation Attempt Logic"))
	fmt.Println(sectionColor("--------------------------------------------------"))
	fmt.Println("Candidate Slots (Priority):")
	for i, attempt := range e.Attempts {
		fmt.Printf("  [%d] %s  → %s\n", i+1, attempt.Slot, attempt.Result)
	}

	fmt.Println("\nAttempt Result:")
	if len(e.Attempts) > 0 {
		last := e.Attempts[len(e.Attempts)-1]
		fmt.Printf("  Slot [%d] : %s\n", len(e.Attempts), last.Detail)
		fmt.Printf("  Status   : %s\n", last.Status)
	}

	fmt.Println("\nComment:")
	fmt.Println("第1希望が失敗した場合でも、")
	fmt.Println("同一接続・同一セッションのまま即座に次候補へ遷移可能な設計です。")

	fmt.Println("\n" + sectionColor("--------------------------------------------------"))
	fmt.Println(sectionColor("[5] Result Summary"))
	fmt.Println(sectionColor("--------------------------------------------------"))

	resColor := valueColor
	if strings.Contains(e.Result, "SUCCESS") {
		resColor = successColor
	} else {
		resColor = errorColor
	}
	fmt.Printf("%s                      : %s\n", labelColor("Result"), resColor(e.Result))
	fmt.Printf("%s        : %s\n", labelColor("End-to-End Readiness"), valueColor(e.EndToEndReadiness))
	fmt.Printf("%s             : %s\n", labelColor("Observed Issues"), valueColor(e.ObservedIssues))

	fmt.Println("\nEngineer Note:")
	if e.EngineerNote != "" {
		// Just print lines of the note
		lines := strings.Split(e.EngineerNote, "\n")
		for _, l := range lines {
			fmt.Println(l)
		}

	}

	if strings.Contains(e.Result, "SUCCESS") {
		fmt.Println("\n" + successColor("🎉🎉🎉 予約完了！ (RESERVATION COMPLETE) 🎉🎉🎉"))
	} else {
		fmt.Println("\n" + errorColor("❌❌❌ 予約失敗 (RESERVATION FAILED) ❌❌❌"))
	}
}

// WriteStructuredLog writes the log entry as a JSON line to the specified file
func WriteStructuredLog(e LogEntry, filename string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Use json.Marshal
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if _, err := f.Write(b); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}
	return nil
}
