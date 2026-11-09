package targets

// All returns every known target in apply order.
func All() []Target {
	return []Target{
		yozakuraTarget(),
		hyprlandTarget(),
		awwwTarget(),
		kittyTarget(),
		gtkTarget(),
		swayncTarget(),
		rofiTarget(),
		neovimTarget(),
		btopTarget(),
		vscodeTarget(),
		qtTarget(),
		fishTarget(),
		starshipTarget(),
		chromeTarget(),
		hyprlockTarget(),
	}
}
