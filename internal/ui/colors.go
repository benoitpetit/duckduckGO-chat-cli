package ui

import "github.com/fatih/color"

// Define a color scheme for the application
var (
	// Main colors
	UserColor    = color.New(color.FgBlue)
	AIColor      = color.New(color.FgGreen)
	SystemColor  = color.New(color.FgCyan)
	WarningColor = color.New(color.FgYellow)
	ErrorColor   = color.New(color.FgRed)
	WhiteColor   = color.New(color.FgWhite)

	// Accents and secondary colors
	PromptColor = color.New(color.FgMagenta)
	MutedColor  = color.New(color.FgHiBlack)
)

// Formatted print functions (without newlines)
func Userf(format string, a ...interface{})    { UserColor.Printf(format, a...) }
func AIf(format string, a ...interface{})      { AIColor.Printf(format, a...) }
func Systemf(format string, a ...interface{})  { SystemColor.Printf(format, a...) }
func Warningf(format string, a ...interface{}) { WarningColor.Printf(format, a...) }
func Errorf(format string, a ...interface{})   { ErrorColor.Printf(format, a...) }
func Whitef(format string, a ...interface{})   { WhiteColor.Printf(format, a...) }
func Promptf(format string, a ...interface{})  { PromptColor.Printf(format, a...) }
func Mutedf(format string, a ...interface{})   { MutedColor.Printf(format, a...) }

// Formatted print functions (with newlines)
func Userln(format string, a ...interface{})    { UserColor.Printf(format+"\n", a...) }
func AIln(format string, a ...interface{})      { AIColor.Printf(format+"\n", a...) }
func Systemln(format string, a ...interface{})  { SystemColor.Printf(format+"\n", a...) }
func Warningln(format string, a ...interface{}) { WarningColor.Printf(format+"\n", a...) }
func Errorln(format string, a ...interface{})   { ErrorColor.Printf(format+"\n", a...) }
func Whiteln(format string, a ...interface{})   { WhiteColor.Printf(format+"\n", a...) }
func Mutedln(format string, a ...interface{})   { MutedColor.Printf(format+"\n", a...) }
