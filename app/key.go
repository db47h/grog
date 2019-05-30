package app

import "fmt"

type Key int

func (k Key) String() string {
	switch k {
	case KeyUnknown:
		return "KeyUnknown"
	case KeySpace:
		return "KeySpace"
	case KeyApostrophe:
		return "KeyApostrophe"
	case KeyComma:
		return "KeyComma"
	case KeyMinus:
		return "KeyMinus"
	case KeyPeriod:
		return "KeyPeriod"
	case KeySlash:
		return "KeySlash"
	case Key0:
		return "Key0"
	case Key1:
		return "Key1"
	case Key2:
		return "Key2"
	case Key3:
		return "Key3"
	case Key4:
		return "Key4"
	case Key5:
		return "Key5"
	case Key6:
		return "Key6"
	case Key7:
		return "Key7"
	case Key8:
		return "Key8"
	case Key9:
		return "Key9"
	case KeySemicolon:
		return "KeySemicolon"
	case KeyEqual:
		return "KeyEqual"
	case KeyA:
		return "KeyA"
	case KeyB:
		return "KeyB"
	case KeyC:
		return "KeyC"
	case KeyD:
		return "KeyD"
	case KeyE:
		return "KeyE"
	case KeyF:
		return "KeyF"
	case KeyG:
		return "KeyG"
	case KeyH:
		return "KeyH"
	case KeyI:
		return "KeyI"
	case KeyJ:
		return "KeyJ"
	case KeyK:
		return "KeyK"
	case KeyL:
		return "KeyL"
	case KeyM:
		return "KeyM"
	case KeyN:
		return "KeyN"
	case KeyO:
		return "KeyO"
	case KeyP:
		return "KeyP"
	case KeyQ:
		return "KeyQ"
	case KeyR:
		return "KeyR"
	case KeyS:
		return "KeyS"
	case KeyT:
		return "KeyT"
	case KeyU:
		return "KeyU"
	case KeyV:
		return "KeyV"
	case KeyW:
		return "KeyW"
	case KeyX:
		return "KeyX"
	case KeyY:
		return "KeyY"
	case KeyZ:
		return "KeyZ"
	case KeyLeftBracket:
		return "KeyLeftBracket"
	case KeyBackslash:
		return "KeyBackslash"
	case KeyRightBracket:
		return "KeyRightBracket"
	case KeyGraveAccent:
		return "KeyGraveAccent"
	case KeyWorld1:
		return "KeyWorld1"
	case KeyWorld2:
		return "KeyWorld2"
	case KeyEscape:
		return "KeyEscape"
	case KeyEnter:
		return "KeyEnter"
	case KeyTab:
		return "KeyTab"
	case KeyBackspace:
		return "KeyBackspace"
	case KeyInsert:
		return "KeyInsert"
	case KeyDelete:
		return "KeyDelete"
	case KeyRight:
		return "KeyRight"
	case KeyLeft:
		return "KeyLeft"
	case KeyDown:
		return "KeyDown"
	case KeyUp:
		return "KeyUp"
	case KeyPageUp:
		return "KeyPageUp"
	case KeyPageDown:
		return "KeyPageDown"
	case KeyHome:
		return "KeyHome"
	case KeyEnd:
		return "KeyEnd"
	case KeyCapsLock:
		return "KeyCapsLock"
	case KeyScrollLock:
		return "KeyScrollLock"
	case KeyNumLock:
		return "KeyNumLock"
	case KeyPrintScreen:
		return "KeyPrintScreen"
	case KeyPause:
		return "KeyPause"
	case KeyF1:
		return "KeyF1"
	case KeyF2:
		return "KeyF2"
	case KeyF3:
		return "KeyF3"
	case KeyF4:
		return "KeyF4"
	case KeyF5:
		return "KeyF5"
	case KeyF6:
		return "KeyF6"
	case KeyF7:
		return "KeyF7"
	case KeyF8:
		return "KeyF8"
	case KeyF9:
		return "KeyF9"
	case KeyF10:
		return "KeyF10"
	case KeyF11:
		return "KeyF11"
	case KeyF12:
		return "KeyF12"
	case KeyF13:
		return "KeyF13"
	case KeyF14:
		return "KeyF14"
	case KeyF15:
		return "KeyF15"
	case KeyF16:
		return "KeyF16"
	case KeyF17:
		return "KeyF17"
	case KeyF18:
		return "KeyF18"
	case KeyF19:
		return "KeyF19"
	case KeyF20:
		return "KeyF20"
	case KeyF21:
		return "KeyF21"
	case KeyF22:
		return "KeyF22"
	case KeyF23:
		return "KeyF23"
	case KeyF24:
		return "KeyF24"
	case KeyKP0:
		return "KeyKP0"
	case KeyKP1:
		return "KeyKP1"
	case KeyKP2:
		return "KeyKP2"
	case KeyKP3:
		return "KeyKP3"
	case KeyKP4:
		return "KeyKP4"
	case KeyKP5:
		return "KeyKP5"
	case KeyKP6:
		return "KeyKP6"
	case KeyKP7:
		return "KeyKP7"
	case KeyKP8:
		return "KeyKP8"
	case KeyKP9:
		return "KeyKP9"
	case KeyKPDecimal:
		return "KeyKPDecimal"
	case KeyKPDivide:
		return "KeyKPDivide"
	case KeyKPMultiply:
		return "KeyKPMultiply"
	case KeyKPSubtract:
		return "KeyKPSubtract"
	case KeyKPAdd:
		return "KeyKPAdd"
	case KeyKPEnter:
		return "KeyKPEnter"
	case KeyKPEqual:
		return "KeyKPEqual"
	case KeyLeftShift:
		return "KeyLeftShift"
	case KeyLeftControl:
		return "KeyLeftControl"
	case KeyLeftAlt:
		return "KeyLeftAlt"
	case KeyLeftSuper:
		return "KeyLeftSuper"
	case KeyRightShift:
		return "KeyRightShift"
	case KeyRightControl:
		return "KeyRightControl"
	case KeyRightAlt:
		return "KeyRightAlt"
	case KeyRightSuper:
		return "KeyRightSuper"
	case KeyMenu:
		return "KeyMenu"
	}
	return fmt.Sprintf("key %d", k)
}