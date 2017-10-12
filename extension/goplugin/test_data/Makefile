SCHEMAS =
BUILD = go build -buildmode=plugin
PLUGINS = ext_good/ext_good.so ext_no_init/ext_no_init.so ext_no_schemas/ext_no_schemas.so

all: $(PLUGINS)
	@echo "finished"

%.so: %.go $(SCHEMAS)
	@echo "building $@..."
	@ $(BUILD) $(BUILD_OPTS) -o $@ $< $(SCHEMAS)
	@echo "$@: `stat --printf="%s" $@` bytes"

.PHONY: clean

clean:
	rm -f $(PLUGINS)
