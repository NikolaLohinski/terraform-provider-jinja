provider "jinja" {
  delimiters {
    variable_start = "{{"
    variable_end   = "}}"
    block_start    = "{%"
    block_end      = "%}"
    comment_start  = "{#"
    comment_end    = "#}"
  }
  strict_undefined  = false
  left_strip_blocks = false
  trim_blocks       = false
}
