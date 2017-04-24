SCHEMAS =
BUILD = go build -buildmode=plugin
PLUGINS = ext_init.so ext_no_init.so

all: $(PLUGINS)
	@echo "finished"

%.so: %.go $(SCHEMAS)
	@echo "building $@..."
	@ $(BUILD) -o $@ $< $(SCHEMAS)
	@echo "$@: `stat --printf="%s" $@` bytes"

.PHONY: clean

clean:
	rm -f $(PLUGINS)
