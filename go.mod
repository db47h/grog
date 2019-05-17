module github.com/db47h/grog

go 1.12

require (
	github.com/db47h/ofs v0.1.2
	github.com/go-gl/glfw v0.0.0-20190409004039-e6da0acd62b1
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/pkg/errors v0.8.1
	golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f // indirect
	golang.org/x/image v0.0.0-20190516052701-61b8692d9a5c
	golang.org/x/net v0.0.0-20190514140710-3ec191127204 // indirect
	golang.org/x/sys v0.0.0-20190516110030-61b9204099cb // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/tools v0.0.0-20190517003510-bffc5affc6df // indirect
)

// this is a hack to disable an annoying gcc warning during development
// and this line should be commented out.
replace github.com/go-gl/glfw => ./vendor/glfw
