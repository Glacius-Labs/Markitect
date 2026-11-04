// Package harness renders what a chat agent of Markitect Desktop starts with: its
// role, its skill and the conversation so far.
package harness

// Document is what a chat agent starts with: Base holds its role and the rules
// the app enforces, Context its skill, and Task the conversation so far.
type Document struct {
	Base, Context, Task string
}
