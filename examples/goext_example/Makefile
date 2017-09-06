BUILD = go build -buildmode=plugin
PLUGINS = example.so

all: $(PLUGINS)
	@echo "finished"

%.so: %.go
	@echo "building $@..."
	@ $(BUILD) -o $@ $<
	@echo "$@: `stat --printf="%s" $@` bytes"

.PHONY: clean

clean:
	rm -f $(PLUGINS)
