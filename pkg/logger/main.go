package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	logProcessSuccessStatus          = "[OK]"
	logProcessFailedStatus           = "[FAILED]"
	logProcessTimeFormat             = "%5.2f sec "
	logProcessInlineProcessMsgFormat = "%s ..."

	logStateRightPartsSeparator = " "
)

var (
	indent = 0

	Out io.Writer
	Err io.Writer
)

type Options struct {
	Out, Err io.Writer
}

func Init(opts Options) {
	Out = os.Stdout
	Err = os.Stderr

	if opts.Out != nil {
		Out = opts.Out
	}

	if opts.Err != nil {
		Err = opts.Err
	}

	if os.Getenv("WERF_LOG_FORCE_COLOR") != "" {
		color.NoColor = false
	}
}

func WithLogIndent(f func() error) error {
	indentUp()
	err := f()
	indentDown()

	return err
}

func LogProcessInline(msg string, processFunc func() error) error {
	return logProcessInlineBase(msg, processFunc, colorizeStep, colorizeSuccess)
}

func LogServiceProcessInline(msg string, processFunc func() error) error {
	return logProcessInlineBase(msg, processFunc, colorizeService, colorizeService)
}

func LogProcess(msg, processMsg string, processFunc func() error) error {
	return logProcessBase(msg, processMsg, processFunc, colorizeStep, colorizeSuccess)
}

func LogServiceProcess(msg, processMsg string, processFunc func() error) error {
	return logProcessBase(msg, processMsg, processFunc, colorizeService, colorizeService)
}

func LogState(msg, state string) {
	logStateBase(msg, state, "", colorizeService, colorizeService)
}

func LogServiceState(msg, state string) {
	logStateBase(msg, state, "", colorizeService, colorizeService)
}

func LogStep(msg string) {
	LogStepF("%s\n", msg)
}

func LogStepF(format string, args ...interface{}) {
	colorizeAndLogF(Out, colorizeStep, format, args...)
}

func LogService(msg string) {
	LogServiceF("%s\n", msg)
}

func LogServiceF(format string, args ...interface{}) {
	colorizeAndLogF(Out, colorizeService, format, args...)
}

func LogInfo(msg string) {
	LogInfoF("%s\n", msg)
}

func LogInfoF(format string, args ...interface{}) {
	colorizeAndLogF(Out, colorizeInfo, format, args...)
}

func LogError(err error) {
	logBase(Err, colorizeBaseF(colorizeWarning, "%s\n", err))
}

func LogWarning(msg string) {
	LogWarningF("%s\n", msg)
}

func LogWarningF(format string, args ...interface{}) {
	colorizeAndLogF(Err, colorizeWarning, format, args...)
}

func colorizeAndLogF(w io.Writer, colorizeFunc func(string) string, format string, args ...interface{}) {
	logF(w, colorizeBaseF(colorizeFunc, format, args...))
}

func colorizeBaseF(colorizeFunc func(string) string, format string, args ...interface{}) string {
	var colorizeLines []string
	lines := strings.Split(fmt.Sprintf(format, args...), "\n")
	for _, line := range lines {
		if line == "" {
			colorizeLines = append(colorizeLines, line)
		} else {
			colorizeLines = append(colorizeLines, colorizeFunc(line))
		}
	}

	return strings.Join(colorizeLines, "\n")
}

func log(w io.Writer, msg string) {
	logF(w, "%s\n", msg)
}

func logF(w io.Writer, format string, args ...interface{}) {
	var linesWithIndent []string
	lines := strings.Split(fmt.Sprintf(format, args...), "\n")
	for _, line := range lines {
		if line == "" {
			linesWithIndent = append(linesWithIndent, line)
		} else {
			linesWithIndent = append(linesWithIndent, fmt.Sprintf("%s%s", logIndent(), line))
		}
	}

	logBase(w, strings.Join(linesWithIndent, "\n"))
}

func logBase(w io.Writer, msg string) {
	fmt.Fprintf(w, msg)
}

func logIndent() string {
	return strings.Repeat("  ", indent)
}

func indentUp() {
	indent += 1
}

func indentDown() {
	if indent == 0 {
		return
	}

	indent -= 1
}

func logProcessInlineBase(processMsg string, processFunc func() error, colorizeProcessMsgFunc, colorizeSuccessFunc func(string) string) error {
	processMsg = fmt.Sprintf(logProcessInlineProcessMsgFormat, processMsg)
	colorizeAndLogF(Out, colorizeProcessMsgFunc, "%s", processMsg)

	resultStatus := logProcessSuccessStatus
	resultColorize := colorizeSuccessFunc
	start := time.Now()

	err := WithLogIndent(processFunc)
	if err != nil {
		resultStatus = logProcessFailedStatus
		resultColorize = colorizeFail
	}

	elapsedSeconds := fmt.Sprintf(logProcessTimeFormat, time.Since(start).Seconds())

	rightPart := prepareLogStateRightPart(processMsg, resultStatus, elapsedSeconds, resultColorize)
	logBase(Out, fmt.Sprintf("%s\n", rightPart))

	return err
}

func logProcessBase(msg, processMsg string, processFunc func() error, colorizeMsgFunc, colorizeSuccessFunc func(string) string) error {
	logStateBase(msg, processMsg, "", colorizeMsgFunc, colorizeSuccessFunc)

	start := time.Now()
	resultStatus := logProcessSuccessStatus

	err := WithLogIndent(processFunc)

	elapsedSeconds := fmt.Sprintf(logProcessTimeFormat, time.Since(start).Seconds())

	if err != nil {
		resultStatus = logProcessFailedStatus
		logStateBase(msg, resultStatus, elapsedSeconds, colorizeFail, colorizeFail)

		return err
	}

	logStateBase(msg, resultStatus, elapsedSeconds, colorizeMsgFunc, colorizeSuccessFunc)

	return nil
}

func logStateBase(msg string, state, time string, colorizeLeftPartFunc, colorizeRightPartFunc func(string) string) {
	leftPart := prepareLogStateLeftPart(msg, state, time, colorizeLeftPartFunc)

	var rightPart string
	if state != "" {
		rightPart = prepareLogStateRightPart(msg, state, time, colorizeRightPartFunc)
	}

	log(Out, fmt.Sprintf("%s%s", leftPart, rightPart))
}

func prepareLogStateLeftPart(msg, state, time string, colorizeFunc func(string) string) string {
	var result string

	spaceLength := availableTerminalLineSpace(state, timeOrStub(time))
	if spaceLength > 0 {
		if spaceLength > len(msg) {
			result = msg
		} else {
			result = msg[0:spaceLength]
		}
	} else {
		return ""
	}

	return colorizeFunc(result)
}

func prepareLogStateRightPart(msg, state, time string, colorizeFunc func(string) string) string {
	var result string
	spaceLength := availableTerminalLineSpace(msg)

	rightPartLength := len(state + timeOrStub(time) + logStateRightPartsSeparator)
	if spaceLength-rightPartLength > 0 {
		result += strings.Repeat(" ", spaceLength-rightPartLength)
	}

	var rightPart []string
	rightPart = append(rightPart, colorizeFunc(state))
	rightPart = append(rightPart, colorizeFunc(time))

	result += strings.Join(rightPart, logStateRightPartsSeparator)

	return result
}

func timeOrStub(time string) string {
	if time == "" {
		return fmt.Sprintf(logProcessTimeFormat, 0.0)
	}

	return time
}

func availableTerminalLineSpace(parts ...string) int {
	logIndentLength := len(logIndent())
	msgsLength := len(strings.Join(parts, " "))

	return TerminalWidth() - logIndentLength - msgsLength
}

func colorizeFail(msg string) string {
	return colorizeWarning(msg)
}

func colorizeSuccess(msg string) string {
	return colorize(msg, color.FgGreen, color.Bold)
}

func colorizeStep(msg string) string {
	return colorize(msg, color.FgYellow, color.Bold)
}

func colorizeService(msg string) string {
	return colorize(msg, color.FgWhite, color.Bold)
}

func colorizeInfo(msg string) string {
	return colorize(msg, color.FgBlue)
}

func colorizeWarning(msg string) string {
	return colorize(msg, color.FgRed, color.Bold)
}

func colorize(msg string, attributes ...color.Attribute) string {
	return color.New(attributes...).Sprint(msg)
}