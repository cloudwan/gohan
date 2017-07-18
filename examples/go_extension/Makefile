SCHEMAS = todo.go todo_related.go
BUILD = go build -buildmode=plugin
PLUGINS = go_extension.so

all: $(PLUGINS)
	@echo "finished"

%.so: %.go $(SCHEMAS)
	@echo "building $@..."
	@ $(BUILD) -o $@ $< $(SCHEMAS)
	@echo "$@: `stat --printf="%s" $@` bytes"

.PHONY: clean

clean:
	rm -f $(PLUGINS)

