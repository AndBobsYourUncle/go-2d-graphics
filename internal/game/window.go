package game

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	_ "image/png"
	"log"
	"math/rand"
	"os"
	"strings"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/go-gl/gl/v2.1/gl"
)

const (
	ObjectCount = 500
)

type Texture struct {
	width  uint16
	height uint16
	u1     float32
	v1     float32
	u2     float32
	v2     float32
}

type Object struct {
	x       int16
	y       int16
	texture Texture
}

var watermelon = Texture{
	width:  64,
	height: 64,
	u1:     0,
	v1:     0,
	u2:     0.5,
	v2:     0.5,
}

var pineapple = Texture{
	width:  64,
	height: 64,
	u1:     0.5,
	v1:     0,
	u2:     1,
	v2:     0.5,
}

var orange = Texture{
	width:  32,
	height: 32,
	u1:     0,
	v1:     0.5,
	u2:     0.25,
	v2:     0.75,
}

var grapes = Texture{
	width:  32,
	height: 32,
	u1:     0.25,
	v1:     0.5,
	u2:     0.5,
	v2:     0.75,
}

var pear = Texture{
	width:  32,
	height: 32,
	u1:     0,
	v1:     0.75,
	u2:     0.25,
	v2:     1,
}

var banana = Texture{
	width:  32,
	height: 32,
	u1:     0.25,
	v1:     0.75,
	u2:     0.5,
	v2:     1,
}

var strawberry = Texture{
	width:  16,
	height: 16,
	u1:     0.5,
	v1:     0.5,
	u2:     0.625,
	v2:     0.625,
}

var raspberry = Texture{
	width:  16,
	height: 16,
	u1:     0.625,
	v1:     0.5,
	u2:     0.75,
	v2:     0.625,
}

var cherries = Texture{
	width:  16,
	height: 16,
	u1:     0.5,
	v1:     0.625,
	u2:     0.625,
	v2:     0.75,
}

var possibleTextures = []Texture{
	watermelon,
	pineapple,
	orange,
	grapes,
	pear,
	banana,
	strawberry,
	cherries,
	raspberry,
}

// we send in the window resolution in the uniforms ww and wh, and then use those to
// determine location based on the window width and height
const (
	vertexShader = `
		#version 330
		uniform float ww;
		uniform float wh;
		layout (location = 0) in vec2 vert;
		layout (location = 1) in vec2 _uv;
		out vec2 uv;
		void main()
		{
			uv = _uv;
			gl_Position = vec4(vert.x / (ww / 2) - 1.0, vert.y / (wh / 2) - 1.0, 0.0, 1.0);
		}
` + "\x00"

	fragmentShader = `
		#version 330
		out vec4 color;
		in vec2 uv;
		uniform sampler2D tex;
		void main()
		{
			color = texture(tex, uv);
		}
` + "\x00"
)

type Window struct {
	title  string
	width  int
	height int
}

type WindowConfig struct {
	Title  string
	Width  int
	Height int
}

func NewWindow(cfg *WindowConfig) (*Window, error) {
	if cfg == nil {
		return nil, errors.New("window missing config")
	}

	return &Window{
		title:  cfg.Title,
		width:  cfg.Width,
		height: cfg.Height,
	}, nil
}

func (w *Window) OpenAndWait() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(w.width, w.height, w.title, nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.CULL_FACE)
	gl.FrontFace(gl.CCW)
	gl.Enable(gl.BLEND)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.SCISSOR_TEST)

	winWidth, winHeight := window.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winWidth), int32(winHeight))

	shaderProgramID, err := getShaderProgramID(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	var textureID uint32

	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	file, err := os.Open("texture.png")
	if err != nil {
		panic(err)
	}

	img, err := png.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Pt(0, 0), draw.Src)

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	vertices := make([]int16, ObjectCount*12)
	uvs := make([]float32, ObjectCount*12)

	objects := make([]Object, 0)

	for i := 0; i < ObjectCount; i++ {
		texIndex := rand.Intn(len(possibleTextures) - 1)

		objects = append(objects, Object{
			x:       int16(rand.Intn(w.width)),
			y:       int16(rand.Intn(w.height)),
			texture: possibleTextures[texIndex],
		})
	}

	for i := 0; i < ObjectCount; i++ {
		// top right
		vertices[i*12] = objects[i].x + int16(objects[i].texture.width)
		vertices[i*12+1] = objects[i].y

		// bottom right
		vertices[i*12+2] = objects[i].x + int16(objects[i].texture.width)
		vertices[i*12+3] = objects[i].y + int16(objects[i].texture.height)

		// top left
		vertices[i*12+4] = objects[i].x
		vertices[i*12+5] = objects[i].y

		// bottom right
		vertices[i*12+6] = objects[i].x + int16(objects[i].texture.width)
		vertices[i*12+7] = objects[i].y + int16(objects[i].texture.height)

		// bottom left
		vertices[i*12+8] = objects[i].x
		vertices[i*12+9] = objects[i].y + int16(objects[i].texture.height)

		// top left
		vertices[i*12+10] = objects[i].x
		vertices[i*12+11] = objects[i].y

		// top right
		uvs[i*12] = objects[i].texture.u2
		uvs[i*12+1] = objects[i].texture.v2

		// bottom right
		uvs[i*12+2] = objects[i].texture.u2
		uvs[i*12+3] = objects[i].texture.v1

		// top left
		uvs[i*12+4] = objects[i].texture.u1
		uvs[i*12+5] = objects[i].texture.v2

		// bottom right
		uvs[i*12+6] = objects[i].texture.u2
		uvs[i*12+7] = objects[i].texture.v1

		// bottom left
		uvs[i*12+8] = objects[i].texture.u1
		uvs[i*12+9] = objects[i].texture.v1

		// top left
		uvs[i*12+10] = objects[i].texture.u1
		uvs[i*12+11] = objects[i].texture.v2
	}

	var vao uint32
	var vbo uint32
	var ubo uint32

	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.GenBuffers(1, &ubo)
	gl.BindVertexArray(vao)

	sizeOfInt16 := int(unsafe.Sizeof(int16(0)))
	sizeOfVertices := int(unsafe.Sizeof(vertices) + unsafe.Sizeof([ObjectCount * 12]int16{}))

	sizeOfFloat32 := int(unsafe.Sizeof(float32(0)))
	sizeOfUvs := int(unsafe.Sizeof(uvs) + unsafe.Sizeof([ObjectCount * 12]float32{}))

	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, sizeOfVertices, gl.Ptr(vertices), gl.DYNAMIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.SHORT, false, 2*int32(sizeOfInt16), gl.PtrOffset(0))

	gl.BindBuffer(gl.ARRAY_BUFFER, ubo)
	gl.BufferData(gl.ARRAY_BUFFER, sizeOfUvs, gl.Ptr(uvs), gl.STATIC_DRAW)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, true, 2*int32(sizeOfFloat32), gl.PtrOffset(0))

	gl.EnableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	gl.UseProgram(shaderProgramID)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.BindVertexArray(vao)

	// send window height and width to shader
	uniformLocationWW := gl.GetUniformLocation(shaderProgramID, gl.Str("ww\x00"))
	uniformLocationWH := gl.GetUniformLocation(shaderProgramID, gl.Str("wh\x00"))

	gl.Uniform1f(uniformLocationWW, float32(w.width))
	gl.Uniform1f(uniformLocationWH, float32(w.height))

	var t float64
	var dt = 0.01
	var accumulator float64
	var frames = 0

	currentTime := glfw.GetTime()
	lastPrinted := currentTime

	for !window.ShouldClose() {
		frames += 1

		newTime := glfw.GetTime()
		frameTime := newTime - currentTime

		currentTime = newTime

		if currentTime-lastPrinted > 1 {
			fmt.Printf("%fms\n", frameTime*1000)
			fmt.Printf("%dfps\n", frames)

			lastPrinted = currentTime
			frames = 0
		}

		accumulator += frameTime

		for accumulator >= dt {
			glfw.PollEvents()

			for i := 0; i < ObjectCount; i++ {
				w.updateObject(i, &objects[i], vertices)
			}

			gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
			gl.BufferSubData(gl.ARRAY_BUFFER, 0, sizeOfVertices, gl.Ptr(vertices))

			accumulator -= dt
			t += dt
		}

		gl.ClearColor(0.2, 0.25, 0.3, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		gl.DrawArrays(gl.TRIANGLES, 0, ObjectCount*6)

		window.SwapBuffers()
	}

	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteBuffers(1, &ubo)
	gl.DeleteTextures(1, &textureID)
	gl.DeleteProgram(shaderProgramID)

	glfw.Terminate()
}

func (w *Window) updateObject(idx int, object *Object, vertices []int16) {
	object.x += int16(rand.Intn(5) - 2)
	object.y += int16(rand.Intn(5) - 2)

	// Simple collision detection. Make sure no sprites exit the window borders
	if object.x < 5 {
		object.x = 5
	}

	if object.x > int16(w.width-5-int(object.texture.width)) {
		object.x = int16(w.width-5) - int16(object.texture.width)
	}

	if object.y < 5 {
		object.y = 5
	}

	if object.y > int16(w.height-5-int(object.texture.height)) {
		object.y = int16(w.height-5) - int16(object.texture.height)
	}

	// top right
	vertices[idx*12] = object.x + int16(object.texture.width)
	vertices[idx*12+1] = object.y

	// bottom right
	vertices[idx*12+2] = object.x + int16(object.texture.width)
	vertices[idx*12+3] = object.y + int16(object.texture.height)

	// top left
	vertices[idx*12+4] = object.x
	vertices[idx*12+5] = object.y

	// bottom right
	vertices[idx*12+6] = object.x + int16(object.texture.width)
	vertices[idx*12+7] = object.y + int16(object.texture.height)

	// bottom left
	vertices[idx*12+8] = object.x
	vertices[idx*12+9] = object.y + int16(object.texture.height)

	// top left
	vertices[idx*12+10] = object.x
	vertices[idx*12+11] = object.y
}

func getShaderProgramID(vertexFile, fragmentFile string) (uint32, error) {
	vertexHandler, err := compileShader(vertexFile, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentFile, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	programID := gl.CreateProgram()

	gl.AttachShader(programID, vertexHandler)
	gl.AttachShader(programID, fragmentShader)
	gl.LinkProgram(programID)

	return programID, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		shaderLog := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(shaderLog))

		return 0, fmt.Errorf("failed to compile %v: %v", source, shaderLog)
	}

	return shader, nil
}
