return {
  {
    "omacom/aether.nvim",
    branch = "v3",
    name = "aether",
    priority = 1000,
    opts = {
      colors = {
        bg         = "#120b18",
        dark_bg    = "#0e0812",
        darker_bg  = "#09060c",
        lighter_bg = "#2a232f",

        fg         = "#f2e7f0",
        dark_fg    = "#b6adb4",
        light_fg   = "#f4ebf2",
        bright_fg  = "#f5edf4",
        muted      = "#a08fa8",

        red        = "#ff5d6c",
        yellow     = "#f2c14e",
        orange     = "#ff7582",
        green      = "#8fd6a8",
        cyan       = "#86d4e6",
        blue       = "#8ba2f1",
        purple     = "#ff9ed2",
        brown      = "#99464e",

        bright_red    = "",
        bright_yellow = "",
        bright_green  = "",
        bright_cyan   = "",
        bright_blue   = "",
        bright_purple = "",

        accent               = "#ff9ed2",
        cursor               = "#ff9ed2",
        foreground           = "#f2e7f0",
        background           = "#120b18",
        selection             = "#2a232f",
        selection_foreground = "#f2e7f0",
        selection_background = "#2a232f",
      },
    },
  },
  {
    "LazyVim/LazyVim",
    opts = {
      colorscheme = "aether",
    },
  },
}
