package govim // import "myitcv.io/govim"

type PluginFactory interface {
	Instance(Client) Plugin
}

type Plugin interface {
	Init()
}

type Client interface {
}
