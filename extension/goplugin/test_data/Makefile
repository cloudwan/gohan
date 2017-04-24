SCHEMAS =
BUILD = go build -buildmode=plugin $(BUILD_OPTS)
PLUGINS = ext_good/ext_good.so ext_no_init/ext_no_init.so ext_no_schemas/ext_no_schemas.so

all: $(PLUGINS)
	@echo "finished"

%.so: %.go $(SCHEMAS)
	@echo "building $@..."
	@ $(BUILD) -o $@ $< $(SCHEMAS)
	@echo "$@: `stat --printf="%s" $@` bytes"

.PHONY: clean

clean:
	rm -f $(PLUGINS)
