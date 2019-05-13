module github.com/db47h/grog

go 1.12

require (
	github.com/db47h/ofs v0.1.2
	github.com/go-gl/glfw v0.0.0-20190409004039-e6da0acd62b1
	github.com/go-gl/mathgl v0.0.0-20190416160123-c4601bc793c7
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/pkg/errors v0.8.1
	golang.org/x/image v0.0.0-20190501045829-6d32002ffd75
)

// this is a hack to disable an annoying gcc warning during development
// and this line should be commented out.
replace github.com/go-gl/glfw => ./vendor/glfw
