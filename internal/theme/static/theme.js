(function () {
  "use strict";

  var STORAGE_KEY = "qtidocs-theme";
  var root = document.documentElement;

  function apply(theme) {
    if (theme === "light" || theme === "dark") {
      root.setAttribute("data-theme", theme);
    }
  }

  try {
    apply(localStorage.getItem(STORAGE_KEY));
  } catch (e) {
    // Storage access can throw (private browsing, blocked storage, etc)
  }

  function effectiveTheme() {
    var explicit = root.getAttribute("data-theme");
    if (explicit === "light" || explicit === "dark") {
      return explicit;
    }
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  document.addEventListener("DOMContentLoaded", function () {
    var versionSelect = document.getElementById("qtidocs-version-select");
    if (versionSelect) {
      versionSelect.addEventListener("change", function () {
        window.location.href = versionSelect.value;
      });
    }

    var button = document.getElementById("qtidocs-theme-toggle");
    if (!button) {
      return;
    }

    function render() {
      var current = effectiveTheme();
      var next = current === "dark" ? "light" : "dark";
      button.setAttribute("aria-label", "Switch to " + next + " mode");
      button.setAttribute("aria-pressed", current === "dark" ? "true" : "false");
    }

    button.addEventListener("click", function () {
      apply(effectiveTheme() === "dark" ? "light" : "dark");
      try {
        localStorage.setItem(STORAGE_KEY, root.getAttribute("data-theme"));
      } catch (e) {
        // Not persisted this time, but the click still took effect
      }
      render();
    });

    render();
  });
})();
