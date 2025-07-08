/*
Copyright © 2025 Roy Sowers <inskribe@inskribestudio.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package apply

//go:generate stringer -type=postStatusEnum

// applyCommandArgs holds parsed CLI arguments for a migration command.
// Note: the meaning of toTag and fromTag depends on the command type:
//   - For "up": fromTag is the lower bound, toTag is the upper bound.
//   - For "down": fromTag is the upper bound, toTag is the lower bound.
type applyCommandArgs struct {
	dryRun               bool     // if true, prints actions without executing them
	PruneNoOp            bool     // if true, skips deltas that are no-ops
	connKey              string   // the environment key to retirve the PostgreSQL connection string. Ignored if connString is passed.
	connString           string   // full PostgreSQL connection string
	toTag                string   // boundary tag (upper for up, lower for down)
	fromTag              string   // boundary tag (lower for up, upper for down)
	cherryPickedVersions []string // specific delta tags to apply instead of a range
}

// deltaRequest defines the range or specific set of deltas to apply.
//
// For an "up" command:
//   - From is the lower bound (inclusive)
//   - To is the upper bound (inclusive)
//
// For a "down" command:
//   - From is the upper bound (inclusive)
//   - To is the lower bound (inclusive)
//
// If Cherries is set, it overrides From/To and applies only the specified tags.
type deltaRequest struct {
	To       *int          // tag boundary (upper for up, lower for down)
	From     *int          // tag boundary (lower for up, upper for down)
	Cherries *map[int]bool // specific delta tags to apply; overrides From/To if set
	LastTag  *int          // indicates that only the last delta should be applied
}

// Represents user input for post command.
type postCmdRequest struct {
	Force bool // allow applying post deltas even if not registered in schemer
}

// postStatusEnum represents the state of a post delta.
//
// Possible values:
//   - 0: NoExist (no associated post delta)
//   - 1: Pending (post delta exists but hasn't been applied)
//   - 2: Applied (post delta has been executed)
type postStatusEnum int

const (
	NoExist postStatusEnum = iota // no associated post delta
	Pending                       // post delta exists but has not been applied
	Applied                       // post delta has been successfully applied
)

// postDelta represents a post-migration delta and its metadata.
type postDelta struct {
	Tag        int            // unique identifier of the delta
	Data       []byte         // raw SQL content of the post delta
	PostStatus postStatusEnum // current post status (e.g., Pending, Applied)
}

// upDelta represents a forward (up) delta and its metadata.
type upDelta struct {
	Tag        int            // unique identifier of the delta
	Data       []byte         // raw SQL content of the up delta
	PostStatus postStatusEnum // post delta status (e.g., NoExist, Pending)
}
