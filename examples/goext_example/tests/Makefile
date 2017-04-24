BUILD = go build -buildmode=plugin
PLUGINS = test_example.so

all: $(PLUGINS)
	@echo "finished"

%.so: %.go
	@echo "building $@..."
	@ $(BUILD) -o $@ $<
	@echo "$@: `stat --printf="%s" $@` bytes"

.PHONY: clean

clean:
	rm -f $(PLUGINS)
