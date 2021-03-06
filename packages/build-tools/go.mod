module github.com/ngld/knossos/packages/build-tools

go 1.15

require (
	github.com/aidarkhanov/nanoid v1.0.8
	github.com/andybalholm/brotli v1.0.1
	github.com/cortesi/modd v0.0.0-20200630120222-8983974e5450
	github.com/go-task/task/v3 v3.2.2
	github.com/golang/protobuf v1.4.3
	github.com/jschaf/pggen v0.0.0-20210208172654-e5703e272221
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db
	github.com/rotisserie/eris v0.5.0
	github.com/rs/zerolog v1.15.0
	github.com/schollz/progressbar/v3 v3.7.4
	github.com/spf13/cobra v1.1.3
	github.com/twitchtv/twirp v7.1.0+incompatible
	github.com/ulikunitz/xz v0.5.10
	go.starlark.net v0.0.0-20210223155950-e043a3d3c984
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	mvdan.cc/sh/v3 v3.3.0-0.dev.0.20210226093739-3d8d47845eeb
)

replace github.com/ngld/knossos/packages/libknossos => ../libknossos

replace github.com/ngld/knossos/packages/libarchive => ../libarchive

replace github.com/ngld/knossos/packages/api => ../api
