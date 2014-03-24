// 14 march 2014

package ui

import (
	"sync"
	"image"
)

// Area represents a blank canvas upon which programs may draw anything and receive arbitrary events from the user.
// An Area has an explicit size, represented in pixels, that may be different from the size shown in its Window; scrollbars are placed automatically should they be needed.
// The coordinate system of an Area always has an origin of (0,0) which maps to the top-left corner; all image.Points and image.Rectangles sent across Area's channels conform to this.
// 
// To handle events to the Area, an Area must be paired with an AreaHandler.
// See AreaHandler for details.
// 
// Do not use an Area if you intend to read text.
// Due to platform differences regarding text input,
// keyboard events have beem compromised in
// such a way that attempting to read Unicode data
// in platform-native ways is painful.
// [Use TextArea instead, providing a TextAreaHandler.]
// 
// To facilitate development and debugging, for the time being, Areas only work on GTK+.
type Area struct {
	lock			sync.Mutex
	created		bool
	sysData		*sysData
	handler		AreaHandler
	initwidth		int
	initheight		int
}

// AreaHandler represents the events that an Area should respond to.
// You are responsible for the thread safety of any members of the actual type that implements ths interface.
// (Having to use this interface does not strike me as being particularly Go-like, but the nature of Paint makes channel-based event handling a non-option; in practice, deadlocks occur.)
type AreaHandler interface {
	// Paint is called when the Area needs to be redrawn.
	// You MUST handle this event, and you MUST return a valid image, otherwise deadlocks and panicking will occur.
	// The image returned must have the same size as rect (but does not have to have the same origin points).
	// Example:
	// 	imgFromFile, _, err := image.Decode(file)
	// 	if err != nil { panic(err) }
	// 	img := image.NewNRGBA(imgFromFile.Rect)
	// 	draw.Draw(img, img.Rect, imgFromFile, image.ZP, draw.Over)
	// 	// ...
	// 	func (h *myAreaHandler) Paint(rect image.Rectangle) *image.NRGBA {
	// 		return img.SubImage(rect).(*image.NRGBA)
	// 	}
	Paint(rect image.Rectangle) *image.NRGBA

	// Mouse is called when the Area receives a mouse event.
	// You are allowed to do nothing in this handler (to ignore mouse events).
	// See MouseEvent for details.
	Mouse(e MouseEvent)

	// Key is called when the Area receives a keyboard event.
	// You are allowed to do nothing except return false in this handler (to ignore keyboard events).
	// Do not do nothing but return true; this may have unintended consequences.
	// See KeyEvent for details.
	Key(e KeyEvent) bool
}

// MouseEvent contains all the information for a mous event sent by Area.Mouse.
// Mouse button IDs start at 1, with 1 being the left mouse button, 2 being the middle mouse button, and 3 being the right mouse button.
// (TODO "If additional buttons are supported, they will be returned with 4 being the first additional button (XBUTTON1 on Windows), 5 being the second (XBUTTON2 on Windows), and so on."?) (TODO get the user-facing name for XBUTTON1/2; find out if there's a way to query available button count)
type MouseEvent struct {
	// Pos is the position of the mouse in the Area at the time of the event.
	// TODO rename to Pt or Point?
	Pos			image.Point

	// If the event was generated by a mouse button being pressed, Down contains the ID of that button.
	// Otherwise, Down contains 0.
	Down		uint

	// If the event was generated by a mouse button being released, Up contains the ID of that button.
	// Otherwise, Up contains 0.
	// If both Down and Up are 0, the event represents mouse movement (with optional held buttons; see below).
	// Down and Up shall not both be nonzero.
	Up			uint

	// If Down is nonzero, Count indicates the number of clicks: 1 for single-click, 2 for double-click.
	// If Count == 2, AT LEAST one event with Count == 1 will have been sent prior.
	// (This is a platform-specific issue: some platforms send one, some send two.)
	Count		uint

	// Modifiers is a bit mask indicating the modifier keys being held during the event.
	Modifiers		Modifiers

	// Held is a slice of button IDs that indicate which mouse buttons are being held during the event.
	// Held will not include Down and Up.
	// (TODO "There is no guarantee that Held is sorted."?)
	Held			[]uint
}

// HeldBits returns Held as a bit mask.
// Bit 0 maps to button 1, bit 1 maps to button 2, etc.
func (e MouseEvent) HeldBits() (h uintptr) {
	for _, x := range e.Held {
		h |= uintptr(1) << (x - 1)
	}
	return h
}

// A KeyEvent represents a keypress in an Area.
// 
// In a perfect world, KeyEvent would be 100% predictable.
// Despite my best efforts to do this, however, the various
// differences in input handling between each backend
// environment makes this completely impossible (I can
// work with two of the three identically, but not all three).
// Keep this in mind, and remember that Areas are not ideal
// for text. For more details, see areaplan.md and the linked
// tweets at the end of that file. If you know a better solution
// than the one I have chosen, please let me know.
// 
// When you are finished processing the incoming event,
// return whether or not you did something in response
// to the given keystroke from your Key() implementation.
// If you send false, you indicate that you did not handle
// the keypress, and that the system should handle it instead.
// (Some systems will stop processing the keyboard event at all
// if you return true unconditionally, which may result in unwanted
// behavior like global task-switching keystrokes not being processed.)
// 
// If a key is pressed that is not supported by ASCII, ExtKey,
// or Modifiers, no KeyEvent will be produced, and package
// ui will act as if false was returned.
type KeyEvent struct {
	// ASCII is a byte representing the character pressed.
	// Despite my best efforts, this cannot be trivialized
	// to produce predictable input rules on all OSs, even if
	// I try to handle physical keys instead of equivalent
	// characters. Therefore, what happens when the user
	// inserts a non-ASCII character is undefined (some systems
	// will give package ui the underlying ASCII key and we
	// return it; other systems do not). This is especially important
	// if the given input method uses Modifiers to enter characters.
	// If the parenthesized rule cannot be followed and the user
	// enters a non-ASCII character, it will be ignored (package ui
	// will act as above regarding keys it cannot handle).
	// In general, alphanumeric characters, ',', '.', '+', '-', and the
	// (space) should be available on all keyboards. Other ASCII
	// whitespace keys mentioned below may be available, but
	// mind layout differences.
	// Whether or not alphabetic characters are uppercase or
	// lowercase is undefined, and cannot be determined solely
	// by examining Modifiers for Shift. Correct code should handle
	// both uppercase and lowercase identically.
	// In addition, ASCII will contain
	// - ' ' (space) if the spacebar was pressed
	// - '\t' if Tab was pressed, regardless of Modifiers
	// - '\n' if any Enter/Return key was pressed, regardless of which
	// - '\b' if the typewriter Backspace key was pressed
	// If this value is zero, see ExtKey.
	ASCII	byte

	// If ASCII is zero, ExtKey contains a predeclared identifier
	// naming an extended key. See ExtKey for details.
	// If both ASCII and ExtKey are zero, a Modifier by itself
	// was pressed. ASCII and ExtKey will not both be nonzero.
	ExtKey		ExtKey

	Modifiers		Modifiers

	// If Up is true, the key was released; if not, the key was pressed.
	// There is no guarantee that all pressed keys shall have
	// corresponding release events (for instance, if the user switches
	// programs while holding the key down, then releases the key).
	// Keys that have been held down are reported as multiple
	// key press events.
	Up			bool
}

// ExtKey represents keys that do not have an ASCII representation.
// There is no way to differentiate between left and right ExtKeys.
type ExtKey uintptr
const (
	Escape ExtKey = iota + 1
	Insert
	Delete
	Home
	End
	PageUp
	PageDown
	Up
	Down
	Left
	Right
	F1		// no guarantee is made that Fn == F1+n in the future
	F2
	F3
	F4
	F5
	F6
	F7
	F8
	F9
	F10
	F11
	F12
	_nextkeys		// for sanity check
)

// Modifiers indicates modifier keys being held during an event.
// There is no way to differentiate between left and right modifier keys.
type Modifiers uintptr
const (
	Ctrl Modifiers = 1 << iota		// the canonical Ctrl keys ([TODO] on Mac OS X, Control on others)
	Alt						// the canonical Alt keys ([TODO] on Mac OS X, Meta on Unix systems, Alt on others)
	Shift						// the Shift keys
	// TODO add Super
)

// NewArea creates a new Area with the given size and handler.
// It panics if handler is nil.
func NewArea(width int, height int, handler AreaHandler) *Area {
	if handler == nil {
		panic("handler passed to NewArea() must not be nil")
	}
	return &Area{
		sysData:		mksysdata(c_area),
		handler:		handler,
		initwidth:		width,
		initheight:		height,
	}
}

// SetSize sets the Area's internal drawing size.
// It has no effect on the actual control size.
func (a *Area) SetSize(width int, height int) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.created {
		a.sysData.setAreaSize(width, height)
		return
	}
	a.initwidth = width
	a.initheight = height
}

func (a *Area) make(window *sysData) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.sysData.handler = a.handler
	err := a.sysData.make("", window)
	if err != nil {
		return err
	}
	a.sysData.setAreaSize(a.initwidth, a.initheight)
	a.created = true
	return nil
}

func (a *Area) setRect(x int, y int, width int, height int, rr *[]resizerequest) {
	*rr = append(*rr, resizerequest{
		sysData:	a.sysData,
		x:		x,
		y:		y,
		width:	width,
		height:	height,
	})
}

func (a *Area) preferredSize() (width int, height int) {
	return a.sysData.preferredSize()
}
