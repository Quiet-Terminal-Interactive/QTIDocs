(function () {
  "use strict";

  var rangeButtons = document.querySelectorAll("[data-qtidocs-range]");
  var viewsEl = document.getElementById("qtidocs-stats-views");
  var uniquesEl = document.getElementById("qtidocs-stats-uniques");
  var pagesEl = document.getElementById("qtidocs-stats-pages");
  var referrersEl = document.getElementById("qtidocs-stats-referrers");
  var statusEl = document.getElementById("qtidocs-stats-status");
  var basePath = document.body.getAttribute("data-qtidocs-base") || "";

  function renderList(el, items, labelKey) {
    el.innerHTML = "";
    if (!items || items.length === 0) {
      var empty = document.createElement("li");
      empty.className = "qtidocs-stats-empty";
      empty.textContent = "No data yet";
      el.appendChild(empty);
      return;
    }
    items.forEach(function (item) {
      var li = document.createElement("li");
      var label = document.createElement("span");
      label.textContent = item[labelKey];
      var count = document.createElement("span");
      count.className = "qtidocs-stats-count";
      count.textContent = item.views;
      li.appendChild(label);
      li.appendChild(count);
      el.appendChild(li);
    });
  }

  function load(range) {
    statusEl.textContent = "Loading…";
    fetch(basePath + "/_qtidocs/stats.json?range=" + encodeURIComponent(range))
      .then(function (res) {
        if (!res.ok) throw new Error("request failed");
        return res.json();
      })
      .then(function (data) {
        statusEl.textContent = "";
        viewsEl.textContent = data.views;
        uniquesEl.textContent = data.uniques;
        renderList(pagesEl, data.top_pages, "path");
        renderList(referrersEl, data.referrers, "referrer");
      })
      .catch(function () {
        statusEl.textContent = "Couldn't load stats.";
      });
  }

  rangeButtons.forEach(function (btn) {
    btn.addEventListener("click", function () {
      rangeButtons.forEach(function (b) {
        b.classList.remove("qtidocs-stats-range-active");
      });
      btn.classList.add("qtidocs-stats-range-active");
      load(btn.getAttribute("data-qtidocs-range"));
    });
  });

  load("7d");
})();
