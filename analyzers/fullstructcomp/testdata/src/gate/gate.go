// Package gate holds the fixtures of the cost gate of rule G12. The
// gate counts the cmpopts.IgnoreFields names that one comparison of the
// whole value needs. A group above the maxignore setting gets no
// report, because the ignore list of the fix would state more than the
// assertions it replaces. The rule reads test files only, so nothing
// here reports.
package gate

// Response is the produced value of most fixtures. It holds two large
// subtrees, so one assertion inside each one leaves the whole of both
// in the ignore list.
type Response struct {
	Pagination Pagination
	First      Segment
	Second     Segment
	Status     string
	Slots      [2]Segment
	Ledger     map[string]Segment
	Notes      []string
	Owner      *Order
}

// Pagination is a small subtree. The anchor of two assertions under it
// sits there, and the cut of that subtree needs one name.
type Pagination struct {
	Page  int
	Size  int
	Total int
}

// Segment is a large subtree. One assertion inside it leaves every
// other field of it in the ignore list.
type Segment struct {
	Amount   int
	Currency string
	Label    string
	Note     string
	Source   string
	Target   string
}

// Summary carries the same field paths as a Response, and it is another
// type. A want value of the type of the base does not stand beside it,
// so such a site is no roundtrip.
type Summary struct {
	First  Segment
	Second Segment
}

// Order is a small produced value. One comparison of the whole of it
// needs one ignore name, so the gate keeps the report.
type Order struct {
	Name  string
	Count int
	State string
}

// Ticket holds a stamp beside two fields of its own.
type Ticket struct {
	Ref   string
	Note  string
	Stamp Stamp
}

// Stamp carries the Equal method that cmp calls. cmp reads no field
// under such a type, so a whole stamp under the anchor costs one ignore
// name. An assertion that names a field of the stamp meets the same
// stop, and the seven fields of the type cost one name there as well.
type Stamp struct {
	Year   int
	Month  int
	Day    int
	Hour   int
	Minute int
	Second int
	Zone   int
}

// Equal compares two stamps.
func (s Stamp) Equal(other Stamp) bool { return s == other }

// Envelope embeds two structures and a defined string. A test writes
// "got.Code" for the field "Header.Code", and the rule resolves the
// real path through the selections of the type checker.
type Envelope struct {
	Meta
	Kind
	Header
	Body string
}

// Meta is the first embedded structure. The search of a promoted field
// reads it and finds no field of the header there.
type Meta struct {
	Trace string
}

// Kind is a defined string, and a structure can embed it. The search of
// a promoted field then reads a type that carries no field at all.
type Kind string

// Header is the second embedded structure.
type Header struct {
	Code   int
	Reason string
}

// Ring holds itself through a pointer. Every level of a chain that
// names the same field again and again costs one ignore name.
type Ring struct {
	Next *Ring
	Name string
}

// Cycle holds itself through a slice, and it holds no field at all. The
// walk that removes the containers of a type never meets a structure
// here, so the bound of that walk ends it.
type Cycle []Cycle

// Loop holds such a type beside two fields of its own.
type Loop struct {
	Name   string
	Count  int
	Cycles Cycle
}

// Point carries its own Equal method, and cmp calls it instead of
// reading the two fields. A comparison of a point states the claims of
// a test that names both fields.
type Point struct {
	X int
	Y int
}

// Equal compares two points.
func (p Point) Equal(other Point) bool { return p == other }

// Circle carries the same method and one field more. cmp skips no field
// of it, so a test that names two of the three fields has no fix.
type Circle struct {
	X int
	Y int
	R int
}

// Equal compares two circles.
func (c Circle) Equal(other Circle) bool { return c == other }

// Record holds an unexported field, and the subtree under it holds
// none. A comparison at that subtree needs no cmp.AllowUnexported.
type Record struct {
	Detail Detail
	tag    int
}

// Detail is the subtree that holds every asserted path of a record.
type Detail struct {
	Code   int
	Reason string
	Extra  string
}

// response returns the produced value of the fixtures.
func response() Response { return Response{Status: "ok"} }

// responsePointer returns the same value through a pointer.
func responsePointer() *Response { return &Response{Status: "ok"} }

// summary returns a value of another type with the same field paths.
func summary() Summary { return Summary{} }

// order returns a small produced value.
func order() Order { return Order{Name: "boot", Count: 3, State: "new"} }

// ticket returns a value that holds a stamp.
func ticket() Ticket { return Ticket{Ref: "boot", Note: "run"} }

// envelope returns a value that carries promoted fields.
func envelope() Envelope { return Envelope{Body: "text"} }

// ring returns a value that holds itself.
func ring() Ring { return Ring{Name: "boot"} }

// loop returns a value that holds a type with no structure under it.
func loop() Loop { return Loop{Name: "boot", Count: 3} }

// point returns a value that answers a comparison itself.
func point() Point { return Point{X: 1, Y: 2} }

// circle returns a value that answers a comparison itself and holds one
// field more than the assertions name.
func circle() Circle { return Circle{X: 1, Y: 2, R: 3} }

// record returns a value that holds an unexported field.
func record() Record { return Record{tag: 1} }
