package plugins

/*
	This `sub_package_plugin.go` file allow you to import other packages.

	Declare the package you want to import above,
	then you can use the package's functions below to initialize the package

	The InitPlugins() function in this file will be automatically run by the program to initialize,
	it's at `utils\internal_plugin\register.go`

	Using this method to add plugins, users need to edit the code, which may not be a good way.

	You can also add an init() function to the file like other single source plugins,
	and then use function from other packages to init, this also allows remove plugin by delete file.

	Example file `plugin_some_plugin.go`

	```

	package plugins

	import "package_path_or_url"

	func init() {
		thatpackage.Init_functions()
	}
	```
*/
func InitPlugins() {
	// saved_message.Init()
}
