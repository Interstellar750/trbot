package plugins

/*
	This `base.go` file is necessary, please do not delete it.

	The InitPlugins() function in this file will be automatically run
	by the program to initialize, its at `utils\internal_plugin\register.go`.

	You can create a single file in this directory to add plugins,
	just create an `init()` function in it and call functions of
	the `plugin_utils` package to register your plugin.

	If your plugin is complex, you may want to split it into multiple
	source code files.

	You can create a folder in this directory with the name of your
	plugin (for example, it called `sticker_downloader`), split the functions
	into multiple source files, and then create a `plugin_sticker_downloader.go`
	file in this directory and fill it with:

```
	package plugins

	import "trle5.xyz/gopkg/trbot/plugins/sticker_downloader"

	func init() {
		sticker_downloader.RegisterFunc()
	}
```

	You need to implement the `RegisterFunc()` function yourself
	The code in it actually calls various functions
	of the `plugin_utils` package to register your plugin.

	You can also put this process into the `init()` function
	of `sticker_downloader` package, but you need to call another export
	function in the `plugin_sticker_downloader.go`` file, otherwise
	the `init()` function in the sticker_downloader package will not be called.
*/
func InitPlugins() {
	// don't add anything here
}
