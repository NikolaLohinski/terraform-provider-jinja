provider "jinja" {
  delimiters {
    // The values below are the defaults
    variable_start = "{{"
    variable_end   = "}}"
    block_start    = "{%"
    block_end      = "%}"
    comment_start  = "{#"
    comment_end    = "#}"
  }
  strict_undefined = true
}
