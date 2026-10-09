(function () {
  "use strict";

  var MAX_RESULTS = 10;
  var EXCERPT_LENGTH = 140;
  var EXCERPT_CONTEXT_BEFORE = 40;
  var SEARCH_DEBOUNCE_MS = 100;

  var input = document.getElementById("qtidocs-search-input");
  var results = document.getElementById("qtidocs-search-results");
  var nav = document.querySelector(".qtidocs-nav");
  if (!input || !results) return;

  var basePath = document.body.getAttribute("data-qtidocs-base") || "";

  function reposition() {
    var rect = input.getBoundingClientRect();
    results.style.left = rect.left + "px";
    results.style.top = rect.bottom + "px";
  }

  var indexPromise = null;

  function loadIndex() {
    if (!indexPromise) {
      indexPromise = fetch(basePath + "/_qtidocs/search-index.json")
        .then(function (res) { return res.json(); })
        .catch(function () { return []; });
    }
    return indexPromise;
  }

  function score(doc, terms) {
    var title = doc.title.toLowerCase();
    var content = doc.content.toLowerCase();
    var total = 0;
    for (var i = 0; i < terms.length; i++) {
      var term = terms[i];
      if (!term) continue;
      if (title.indexOf(term) !== -1) total += 10;
      if (content.indexOf(term) !== -1) total += 1;
    }
    return total;
  }

  function excerpt(content, term) {
    var lower = content.toLowerCase();
    var at = term ? lower.indexOf(term) : -1;
    var start = at === -1 ? 0 : Math.max(0, at - EXCERPT_CONTEXT_BEFORE);
    var snippet = content.slice(start, start + EXCERPT_LENGTH).trim();
    return (start > 0 ? "…" : "") + snippet + (start + EXCERPT_LENGTH < content.length ? "…" : "");
  }

  function render(docs, query) {
    results.innerHTML = "";
    if (!query) {
      results.hidden = true;
      return;
    }
    reposition();

    var terms = query.toLowerCase().split(/\s+/).filter(Boolean);
    var matches = docs
      .map(function (doc) { return { doc: doc, score: score(doc, terms) }; })
      .filter(function (m) { return m.score > 0; })
      .sort(function (a, b) { return b.score - a.score; })
      .slice(0, MAX_RESULTS);

    if (matches.length === 0) {
      var empty = document.createElement("li");
      empty.className = "qtidocs-search-empty";
      empty.textContent = "No results";
      results.appendChild(empty);
      results.hidden = false;
      return;
    }

    matches.forEach(function (m) {
      var li = document.createElement("li");
      var a = document.createElement("a");
      a.href = basePath + "/" + m.doc.path;
      a.textContent = m.doc.title;
      var p = document.createElement("p");
      p.textContent = excerpt(m.doc.content, terms[0]);
      li.appendChild(a);
      li.appendChild(p);
      results.appendChild(li);
    });
    results.hidden = false;
  }

  var debounceTimer = null;
  input.addEventListener("input", function () {
    var query = input.value.trim();
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function () {
      loadIndex().then(function (docs) { render(docs, query); });
    }, SEARCH_DEBOUNCE_MS);
  });

  input.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      input.value = "";
      results.hidden = true;
    }
  });

  document.addEventListener("click", function (e) {
    if (e.target !== input && !results.contains(e.target)) {
      results.hidden = true;
    }
  });

  window.addEventListener("resize", function () {
    if (!results.hidden) reposition();
  });
  window.addEventListener("scroll", function () {
    if (!results.hidden) reposition();
  }, true);
  if (nav) {
    nav.addEventListener("scroll", function () {
      if (!results.hidden) reposition();
    });
  }
})();
