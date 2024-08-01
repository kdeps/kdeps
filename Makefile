# Directory containing the .pkl files
PKL_DIR := pkl
# Directory where the generated output will be stored
OUTPUT_DIR := gen
# Command to process .pkl files
GEN_COMMAND := pkl-gen-go
# Get the current directory
CURRENT_DIR := $(shell pwd)
# Collect all .pkl files in PKL_DIR
PKL_FILES := $(wildcard $(PKL_DIR)/**/*.pkl)

# Generate output files in OUTPUT_DIR
generate:
	@rm -rf $(OUTPUT_DIR)
	@mkdir -p $(OUTPUT_DIR).tmp
	@for pkl in $(PKL_FILES); do \
		$(GEN_COMMAND) $$pkl --output-path $(OUTPUT_DIR).tmp; \
	done; \
	mv $(OUTPUT_DIR).tmp/github.com/kdeps/kdeps/gen $(OUTPUT_DIR); \
	rm -rf $(OUTPUT_DIR).tmp;
