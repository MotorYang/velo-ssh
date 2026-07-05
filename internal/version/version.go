package version

const Fallback = "v1.0.0.26070601"

var Current = Fallback

func String() string {
	if Current == "" {
		return Fallback
	}
	return Current
}
