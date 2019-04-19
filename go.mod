module github.com/db47h/grog

go 1.12

require github.com/go-gl/glfw v0.0.0-20190409004039-e6da0acd62b1

// this is a hack to disable an annoying gcc warning during development
// and this line should be commented out.
replace github.com/go-gl/glfw => ./vendor/glfw
