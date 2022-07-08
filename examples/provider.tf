provider "jinja" {
  delimiters {
    // The following values are the defaults
    variable_start = "{{"
    variable_end   = "}}"
    block_start    = "{%"
    block_end      = "%}"
    comment_start  = "{#"
    comment_end    = "#}"
  }
}
