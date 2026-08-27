/* Progressive enhancement for the public gallery: a lightbox over the photo
   grid (keyboard, swipe, pinch/zoom, prefetch), copy-link and theme toggle.
   Everything works without this file; it only makes it nicer. */
(function () {
  "use strict";

  // --- theme toggle (persisted per visitor) ---
  var root = document.documentElement;
  try {
    var stored = localStorage.getItem("theme");
    if (stored === "dark" || stored === "light") root.setAttribute("data-theme", stored);
  } catch (e) {}
  document.addEventListener("click", function (ev) {
    var t = ev.target.closest("[data-theme-toggle]");
    if (!t) return;
    var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
    root.setAttribute("data-theme", next);
    try { localStorage.setItem("theme", next); } catch (e) {}
  });

  // --- copy link ---
  document.addEventListener("click", function (ev) {
    var b = ev.target.closest("[data-copy-link]");
    if (!b) return;
    ev.preventDefault();
    var url = b.getAttribute("data-copy-link") || location.href;
    if (navigator.share && /Mobi|Android/i.test(navigator.userAgent)) {
      navigator.share({ url: url }).catch(function () {});
      return;
    }
    navigator.clipboard.writeText(url).then(function () {
      var old = b.textContent;
      b.textContent = "✓";
      setTimeout(function () { b.textContent = old; }, 1200);
    });
  });

  // --- lightbox ---
  // The grid is the source of truth: every tile carries data-lb-src (2048),
  // data-lb-href (detail URL) and data-lb-caption. Load-more pages append
  // tiles, so the list is read live on every navigation.
  var lb = null, current = null, scale = 1, tx = 0, ty = 0;

  function tiles() { return Array.prototype.slice.call(document.querySelectorAll("[data-lb-src]")); }

  function open(tile, push) {
    if (!lb) {
      lb = document.createElement("div");
      lb.className = "lightbox";
      lb.innerHTML = '<div class="lb-nav lb-prev" aria-label="previous"></div><img alt=""><div class="lb-nav lb-next" aria-label="next"></div><div class="lb-close" aria-label="close">×</div><div class="lb-caption"></div>';
      document.body.appendChild(lb);
      document.body.style.overflow = "hidden";
      lb.querySelector(".lb-prev").addEventListener("click", function () { step(-1); });
      lb.querySelector(".lb-next").addEventListener("click", function () { step(1); });
      lb.querySelector(".lb-close").addEventListener("click", close);
      lb.querySelector("img").addEventListener("click", toggleZoom);
      bindGestures(lb);
    }
    current = tile;
    scale = 1; tx = 0; ty = 0;
    var img = lb.querySelector("img");
    img.style.transform = "";
    img.src = tile.getAttribute("data-lb-src");
    img.alt = tile.getAttribute("data-lb-caption") || "";
    lb.querySelector(".lb-caption").textContent = tile.getAttribute("data-lb-caption") || "";
    if (push) history.pushState({ lb: tile.getAttribute("data-lb-href") }, "", tile.getAttribute("data-lb-href"));
    prefetch(1); prefetch(-1);
  }

  function neighbour(dir) {
    var all = tiles(), i = all.indexOf(current);
    if (i < 0) return null;
    var n = all[i + dir];
    if (!n && dir > 0) {
      var more = document.querySelector("[data-load-more]");
      if (more && window.htmx) htmx.trigger(more, "click");
    }
    return n || null;
  }

  function prefetch(dir) {
    var n = neighbour(dir);
    if (n) { var i = new Image(); i.src = n.getAttribute("data-lb-src"); }
  }

  function step(dir) {
    var n = neighbour(dir);
    if (n) open(n, true);
  }

  function close() {
    if (!lb) return;
    lb.remove(); lb = null; current = null;
    document.body.style.overflow = "";
    var back = document.querySelector("[data-lb-return]");
    history.pushState({}, "", back ? back.getAttribute("data-lb-return") : location.pathname.replace(/\/p\/[^/]+$/, "/photos"));
  }

  function toggleZoom(ev) {
    var img = lb.querySelector("img");
    if (scale > 1) { scale = 1; tx = 0; ty = 0; }
    else {
      scale = 2.5;
      var r = img.getBoundingClientRect();
      tx = (r.width / 2 - (ev.clientX - r.left)) * (scale - 1);
      ty = (r.height / 2 - (ev.clientY - r.top)) * (scale - 1);
    }
    apply();
  }

  function apply() {
    lb.querySelector("img").style.transform = "translate(" + tx + "px," + ty + "px) scale(" + scale + ")";
  }

  function bindGestures(el) {
    var startX = 0, startY = 0, startDist = 0, startScale = 1, panning = false, lastX = 0, lastY = 0;
    el.addEventListener("touchstart", function (e) {
      if (e.touches.length === 2) {
        startDist = dist(e.touches); startScale = scale;
      } else if (e.touches.length === 1) {
        startX = lastX = e.touches[0].clientX; startY = lastY = e.touches[0].clientY; panning = scale > 1;
      }
    }, { passive: true });
    el.addEventListener("touchmove", function (e) {
      if (e.touches.length === 2) {
        scale = Math.min(5, Math.max(1, startScale * dist(e.touches) / startDist)); apply();
      } else if (panning && e.touches.length === 1) {
        tx += e.touches[0].clientX - lastX; ty += e.touches[0].clientY - lastY;
        lastX = e.touches[0].clientX; lastY = e.touches[0].clientY; apply();
      }
    }, { passive: true });
    el.addEventListener("touchend", function (e) {
      if (panning || e.changedTouches.length !== 1 || scale > 1) return;
      var dx = e.changedTouches[0].clientX - startX, dy = e.changedTouches[0].clientY - startY;
      if (Math.abs(dx) > 50 && Math.abs(dy) < 80) step(dx < 0 ? 1 : -1);
      else if (dy > 90 && Math.abs(dx) < 60) close();
    });
  }

  function dist(t) { var dx = t[0].clientX - t[1].clientX, dy = t[0].clientY - t[1].clientY; return Math.sqrt(dx * dx + dy * dy); }

  document.addEventListener("click", function (ev) {
    var tile = ev.target.closest("[data-lb-src]");
    if (!tile || ev.metaKey || ev.ctrlKey || ev.shiftKey) return;
    ev.preventDefault();
    open(tile, true);
  });

  document.addEventListener("keydown", function (ev) {
    if (!lb) {
      // detail page: arrows follow the prev/next links
      if (ev.key === "ArrowRight") { var n = document.querySelector("[data-nav-next]"); if (n) location.href = n.href; }
      if (ev.key === "ArrowLeft") { var p = document.querySelector("[data-nav-prev]"); if (p) location.href = p.href; }
      return;
    }
    if (ev.key === "Escape") close();
    if (ev.key === "ArrowRight") step(1);
    if (ev.key === "ArrowLeft") step(-1);
  });

  window.addEventListener("popstate", function () {
    if (lb) { lb.remove(); lb = null; current = null; document.body.style.overflow = ""; }
  });
})();
