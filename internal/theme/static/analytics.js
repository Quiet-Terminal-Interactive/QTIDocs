(function () {
  "use strict";

  var basePath = document.body.getAttribute("data-qtidocs-base") || "";
  var payload = JSON.stringify({
    path: window.location.pathname,
    referrer: document.referrer || ""
  });
  var url = basePath + "/_qtidocs/collect";

  try {
    if (navigator.sendBeacon) {
      navigator.sendBeacon(url, new Blob([payload], { type: "application/json" }));
    } else {
      fetch(url, { method: "POST", body: payload, keepalive: true }).catch(function () { });
    }
  } catch (e) {
    // Collection is best-effort
  }
})();
