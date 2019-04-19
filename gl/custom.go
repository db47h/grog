package gl

/*
#cgo !gles1,!gles2 CFLAGS: -DGROG_USE_GL=1
#cgo gles1 gles2 CFLAGS: -DGROG_USE_GLES=1

#ifdef GROG_USE_GL
#include <GL/gl.h>
#else
#include <GLES2/gl2.h>
#endif

const char *grog_glVersion() {
	return glGetString(GL_VERSION);
}
*/
import "C"

func Version() string {
	return C.GoString(C.grog_glVersion())
}
