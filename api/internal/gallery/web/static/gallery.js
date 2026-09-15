/* Progressive enhancement for the public gallery: wheel/pinch zoom on the
   detail hero and its fullscreen overlay, Ctrl+wheel grid sizing, facet
   search, copy-link and theme toggle. Everything works without this file; it
   only makes it nicer. */
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

  // --- search typeahead: close on blur/escape, arrow keys walk the rows ---
  document.addEventListener("keydown", function (ev) {
    var form = ev.target.closest && ev.target.closest("[data-search]");
    if (!form) return;
    var list = form.querySelector("[data-suggest]"), rows = list.querySelectorAll("a"), i = Array.prototype.indexOf.call(rows, list.querySelector("a.active"));
    if (ev.key === "Escape") { list.innerHTML = ""; return; }
    if (!rows.length) return;
    if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
      ev.preventDefault();
      i = (i + (ev.key === "ArrowDown" ? 1 : -1) + rows.length) % rows.length;
      rows.forEach(function (r, k) { r.classList.toggle("active", k === i); });
    } else if (ev.key === "Enter" && i >= 0) {
      ev.preventDefault();
      location.href = rows[i].href;
    }
  });
  document.addEventListener("focusout", function (ev) {
    var form = ev.target.closest && ev.target.closest("[data-search]");
    if (form && !form.contains(ev.relatedTarget)) setTimeout(function () { form.querySelector("[data-suggest]").innerHTML = ""; }, 150);
  });

  // --- facet groups: search across every value, "+N" unfolds the rest ---
  document.addEventListener("input", function (ev) {
    var input = ev.target.closest("[data-facet-search]");
    if (!input) return;
    var q = input.value.trim().toLowerCase(), group = input.closest("[data-facet]");
    group.querySelectorAll("a[data-label]").forEach(function (a) {
      a.hidden = q ? a.getAttribute("data-label").indexOf(q) < 0 : a.hasAttribute("data-more") && !group.hasAttribute("data-open");
    });
  });
  document.addEventListener("click", function (ev) {
    var b = ev.target.closest("[data-facet-more]");
    if (!b) return;
    var group = b.closest("[data-facet]"), open = group.toggleAttribute("data-open");
    group.querySelectorAll("a[data-more]").forEach(function (a) { a.hidden = !open; });
    b.textContent = open ? "−" : "+" + group.querySelectorAll("a[data-more]").length;
  });

  // --- zoom engine (same math as the SPA's util/zoom.ts) ---
  // Wheel zooms about the cursor, drag pans; offsets are clamped so the image
  // keeps covering the viewport, widened by half a viewport of slack per axis
  // so any edge can be dragged to the centre. `onOut` fires when a hard scroll
  // (fast, repeated wheel-out) hits a fitted image; the image dips a little
  // meanwhile so the gesture is visible.
  var MAX_ZOOM = 8, PAN_SLACK = 0.5, OUT_THRESHOLD = 900, OUT_GAP = 160;

  function clampOffset(value, viewport, scaled) {
    var slack = viewport * PAN_SLACK;
    var lo = Math.min(0, viewport - scaled) - slack, hi = Math.max(0, viewport - scaled) + slack;
    return Math.min(hi, Math.max(lo, value));
  }

  function tiles() { return Array.prototype.slice.call(document.querySelectorAll("[data-lb-src]")); }

  // `img` is the transformed node — the hero's stage wrapper (so the hi-res
  // overlay below rides along) or the overlay's <img>. hiresSrc is fetched on
  // first zoom and revealed once loaded, exactly covering the fitted image.
  function zoomer(el, img, onOut, touch, hiresSrc) {
    var scale = 1, tx = 0, ty = 0, out = 0, outAt = 0, outTimer = null, dragged = false, hires = null;

    function loadHires() {
      if (hires || !hiresSrc) return;
      hires = document.createElement("img");
      hires.alt = ""; hires.draggable = false;
      hires.className = "hires";
      hires.addEventListener("load", function () { hires.classList.add("ready"); });
      hires.src = hiresSrc;
      img.appendChild(hires);
    }

    function viewport() { return { w: el.clientWidth, h: el.clientHeight }; }
    // untransformed layout box of the img; offset* ignores the transform, so a
    // transition in flight never skews the math
    function base() { return { left: img.offsetLeft, top: img.offsetTop, w: img.offsetWidth, h: img.offsetHeight }; }

    function apply() {
      img.style.transform = "translate(" + tx + "px," + ty + "px) scale(" + scale + ")";
      el.classList.toggle("zoomed", scale > 1);
      if (scale > 1) loadHires();
    }

    function zoomAt(clientX, clientY, target) {
      var next = Math.min(MAX_ZOOM, Math.max(1, target));
      if (next === 1) { scale = 1; tx = 0; ty = 0; apply(); return; }
      var r = el.getBoundingClientRect(), b = base(), v = viewport();
      var px = clientX - r.left - b.left, py = clientY - r.top - b.top;
      var nx = px - ((px - tx) / scale) * next, ny = py - ((py - ty) / scale) * next;
      scale = next;
      tx = clampOffset(b.left + nx, v.w, b.w * scale) - b.left;
      ty = clampOffset(b.top + ny, v.h, b.h * scale) - b.top;
      apply();
    }

    function panBy(dx, dy) {
      var b = base(), v = viewport();
      tx = clampOffset(b.left + tx + dx, v.w, b.w * scale) - b.left;
      ty = clampOffset(b.top + ty + dy, v.h, b.h * scale) - b.top;
      apply();
    }

    function settle() {
      out = 0;
      img.style.transform = "";
      el.classList.remove("out");
    }

    el.addEventListener("wheel", function (ev) {
      ev.preventDefault();
      if (scale > 1 || ev.deltaY < 0) { zoomAt(ev.clientX, ev.clientY, scale * Math.exp(-ev.deltaY * 0.002)); return; }
      if (!onOut) return;
      // fitted and scrolling out: accumulate a burst, dip the image, leave when it's hard enough
      var now = Date.now();
      out = now - outAt < OUT_GAP ? out + ev.deltaY : ev.deltaY;
      outAt = now;
      clearTimeout(outTimer);
      if (out >= OUT_THRESHOLD) { onOut(); return; }
      el.classList.add("out");
      img.style.transform = "scale(" + (1 - 0.12 * out / OUT_THRESHOLD) + ")";
      outTimer = setTimeout(settle, OUT_GAP * 2);
    }, { passive: false });

    var id = null, lastX = 0, lastY = 0;
    el.addEventListener("pointerdown", function (e) {
      if (e.pointerType !== "mouse" || e.button !== 0 || scale === 1 || !e.target.closest("img")) return;
      e.preventDefault();
      id = e.pointerId; lastX = e.clientX; lastY = e.clientY; dragged = false;
      el.classList.add("panning"); el.setPointerCapture(id);
    });
    el.addEventListener("pointermove", function (e) {
      if (e.pointerId !== id) return;
      var dx = e.clientX - lastX, dy = e.clientY - lastY;
      if (Math.abs(dx) + Math.abs(dy) > 2) dragged = true;
      lastX = e.clientX; lastY = e.clientY;
      panBy(dx, dy);
    });
    function up(e) { if (e.pointerId === id) { id = null; el.classList.remove("panning"); } }
    el.addEventListener("pointerup", up);
    el.addEventListener("pointercancel", up);

    img.addEventListener("click", function (ev) {
      if (dragged) { dragged = false; return; }
      zoomAt(ev.clientX, ev.clientY, scale > 1 ? 1 : 2.5);
    });

    // touch: pinch about the midpoint, one finger pans while zoomed (only in
    // the overlay — the hero must not steal page scrolling on phones)
    var startDist = 0, startScale = 1, tLastX = 0, tLastY = 0;
    if (touch) el.addEventListener("touchstart", function (e) {
      if (e.touches.length === 2) { startDist = dist(e.touches); startScale = scale; }
      else if (e.touches.length === 1) { tLastX = e.touches[0].clientX; tLastY = e.touches[0].clientY; }
    }, { passive: true });
    if (touch) el.addEventListener("touchmove", function (e) {
      if (e.touches.length === 2) {
        zoomAt((e.touches[0].clientX + e.touches[1].clientX) / 2, (e.touches[0].clientY + e.touches[1].clientY) / 2, startScale * dist(e.touches) / startDist);
      } else if (scale > 1 && e.touches.length === 1) {
        panBy(e.touches[0].clientX - tLastX, e.touches[0].clientY - tLastY);
        tLastX = e.touches[0].clientX; tLastY = e.touches[0].clientY;
      }
    }, { passive: true });

    return {
      reset: function () { scale = 1; tx = 0; ty = 0; apply(); },
      zoomed: function () { return scale > 1; },
      by: function (factor) { var r = el.getBoundingClientRect(); zoomAt(r.left + r.width / 2, r.top + r.height / 2, scale * factor); },
    };
  }

  function dist(t) { var dx = t[0].clientX - t[1].clientX, dy = t[0].clientY - t[1].clientY; return Math.sqrt(dx * dx + dy * dy); }

  // --- detail page hero: zoom in place; a hard scroll-out returns to the grid.
  // Prev/next swap #detail via HTMX, so everything hero-bound re-initialises
  // per swap (and after a history restore, which revives a dead snapshot).
  var hero = null;
  function initHero() {
    hero = document.querySelector("[data-hero]");
    if (!hero) return;
    zoomer(hero, hero.firstElementChild, function () {
      var back = document.querySelector("[data-back]");
      if (back) location.href = back.href;
    }, false, hero.getAttribute("data-hires"));
    // warm the neighbours once the hero is in: their page HTML and the same
    // srcset candidate the next hero will pick, so stepping is a cache hit
    var warm = function () {
      [navLink(-1), navLink(1)].forEach(function (n) {
        if (!n) return;
        var l = document.createElement("link"); l.rel = "prefetch"; l.href = n.href; document.head.appendChild(l);
        var i = new Image(); i.sizes = "100vw"; i.srcset = n.getAttribute("data-srcset") || "";
      });
    };
    var heroImg = hero.querySelector("img");
    if (heroImg.complete) warm(); else heroImg.addEventListener("load", warm, { once: true });
  }
  initHero();
  document.body.addEventListener("htmx:afterSettle", function (ev) {
    if (!ev.detail.target || ev.detail.target.id !== "detail") return;
    initHero();
    if (lb) { closeFullscreen(); openFullscreen(); }
  });
  document.body.addEventListener("htmx:historyRestore", function () {
    document.querySelectorAll(".lightbox").forEach(function (e) { e.remove(); });
    lb = null; lbZoom = null; document.body.style.overflow = "";
    closeMenu();
    initHero();
  });

  // --- right-click on the image: one entry, the EXIF-backed download (the
  // browser's "save image" would grab a bare rendition or the raw original)
  var menu = null;
  document.addEventListener("contextmenu", function (ev) {
    var img = ev.target.closest("[data-hero] img, .lightbox img");
    if (!img || !hero) return;
    ev.preventDefault();
    closeMenu();
    menu = document.createElement("div");
    menu.className = "ctx-menu";
    var a = document.createElement("a");
    a.href = hero.getAttribute("data-download");
    a.rel = "nofollow";
    a.textContent = "⤓ " + (hero.getAttribute("data-download-label") || "Download");
    menu.appendChild(a);
    document.body.appendChild(menu);
    menu.style.left = Math.min(ev.clientX, window.innerWidth - menu.offsetWidth - 8) + "px";
    menu.style.top = Math.min(ev.clientY, window.innerHeight - menu.offsetHeight - 8) + "px";
  });
  function closeMenu() { if (menu) { menu.remove(); menu = null; } }
  document.addEventListener("click", closeMenu);
  document.addEventListener("keydown", function (ev) { if (ev.key === "Escape") closeMenu(); });
  window.addEventListener("blur", closeMenu);

  // --- fullscreen lightbox over the detail page (#fs survives reload and
  // prev/next navigation; swipe/arrows follow the detail links) ---
  var lb = null, lbZoom = null;

  function openFullscreen() {
    if (lb || !hero) return;
    lb = document.createElement("div");
    lb.className = "lightbox";
    lb.innerHTML = '<div class="lb-nav lb-prev" aria-label="previous"></div><div class="lb-stage"><img alt="" draggable="false"></div><div class="lb-nav lb-next" aria-label="next"></div><div class="lb-zoom"><button type="button" data-zoom="out" aria-label="zoom out">−</button><button type="button" data-zoom="in" aria-label="zoom in">+</button></div><div class="lb-close" aria-label="close">×</div><div class="lb-caption"></div>';
    document.body.appendChild(lb);
    document.body.style.overflow = "hidden";
    var img = lb.querySelector("img");
    img.src = hero.getAttribute("data-lb-src");
    img.alt = hero.getAttribute("data-lb-caption") || "";
    lb.querySelector(".lb-caption").textContent = img.alt;
    lb.querySelector(".lb-prev").addEventListener("click", function () { step(-1); });
    lb.querySelector(".lb-next").addEventListener("click", function () { step(1); });
    lb.querySelector(".lb-close").addEventListener("click", closeFullscreen);
    lbZoom = zoomer(lb, lb.querySelector(".lb-stage"), closeFullscreen, true, hero.getAttribute("data-hires"));
    bindSwipe(lb);
    if (location.hash !== "#fs") history.replaceState(null, "", "#fs");
  }

  function closeFullscreen() {
    if (!lb) return;
    lb.remove(); lb = null; lbZoom = null;
    document.body.style.overflow = "";
    if (location.hash === "#fs") history.replaceState(null, "", location.pathname + location.search);
  }

  function navLink(dir) { return document.querySelector(dir > 0 ? "[data-nav-next]" : "[data-nav-prev]"); }

  function step(dir) {
    var n = navLink(dir);
    if (!n) return;
    if (window.htmx) n.click(); else location.href = n.href + (lb ? "#fs" : "");
  }

  function bindSwipe(el) {
    var startX = 0, startY = 0;
    el.addEventListener("touchstart", function (e) {
      if (e.touches.length === 1) { startX = e.touches[0].clientX; startY = e.touches[0].clientY; }
    }, { passive: true });
    el.addEventListener("touchend", function (e) {
      if (e.changedTouches.length !== 1 || lbZoom.zoomed()) return;
      var dx = e.changedTouches[0].clientX - startX, dy = e.changedTouches[0].clientY - startY;
      if (Math.abs(dx) > 50 && Math.abs(dy) < 80) step(dx < 0 ? 1 : -1);
      else if (dy > 90 && Math.abs(dx) < 60) closeFullscreen();
    });
  }

  document.addEventListener("click", function (ev) {
    if (ev.target.closest("[data-fullscreen]")) openFullscreen();
  });
  if (hero && location.hash === "#fs") openFullscreen();

  document.addEventListener("keydown", function (ev) {
    if (ev.key === "Escape") closeFullscreen();
    if (ev.key === "ArrowRight") step(1);
    if (ev.key === "ArrowLeft") step(-1);
  });

  // --- Ctrl+wheel: grid tile size → lightbox → image zoom, one continuous axis ---
  var TILE_MIN = 80, TILE_DEFAULT = 260;
  function grids() { return Array.prototype.slice.call(document.querySelectorAll(".grid-justified")); }
  function tileSize() {
    try { var v = parseFloat(localStorage.getItem("tile")); if (v >= TILE_MIN) return v; } catch (e) {}
    return TILE_DEFAULT;
  }
  function setTile(px) {
    grids().forEach(function (g) { g.style.setProperty("--tile", px + "px"); });
    try { localStorage.setItem("tile", String(Math.round(px))); } catch (e) {}
  }

  // FLIP the reflow: tiles near the viewport glide from where they are drawn
  // right now (mid-animation included) to their new layout box.
  function flip(mutate) {
    var span = window.innerHeight, before = [];
    tiles().forEach(function (t) {
      var r = t.getBoundingClientRect();
      if (r.bottom > -span && r.top < 2 * span) before.push({ el: t, r: r });
    });
    mutate();
    before.forEach(function (b) {
      b.el.getAnimations().forEach(function (a) { a.cancel(); });
      var n = b.el.getBoundingClientRect();
      if (!n.width || !n.height) return;
      var from = "translate(" + (b.r.left - n.left) + "px," + (b.r.top - n.top) + "px) scale(" + b.r.width / n.width + "," + b.r.height / n.height + ")";
      b.el.style.transformOrigin = "0 0";
      b.el.animate([{ transform: from }, { transform: "none" }], { duration: 260, easing: "cubic-bezier(0.2, 0, 0, 1)" });
    });
  }
  if (grids().length) setTile(tileSize());

  // --- return from a detail page: #p-<id> may sit on a page not loaded yet ---
  function revealHashTile() {
    var m = /^#p-(.+)$/.exec(location.hash);
    if (!m) return;
    var t = document.getElementById("p-" + m[1]);
    if (t) { t.scrollIntoView({ block: "center" }); return; }
    var more = document.querySelector("[data-load-more]");
    if (more && window.htmx) htmx.trigger(more, "click");
  }
  if (grids().length) {
    revealHashTile();
    document.body.addEventListener("htmx:afterSettle", function () { if (!document.getElementById(location.hash.slice(1))) revealHashTile(); });
  }


  // one step of the grid axis: factor > 1 grows tiles, anchored on `tile`
  // (the one under the cursor, or the first visible one for the buttons)
  function gridZoom(factor, tile) {
    var grid = tile && tile.closest(".grid-justified") || grids()[0];
    if (!grid) return;
    var max = grid.clientWidth, cur = tileSize();
    if (tile && factor > 1 && (cur >= max || tile.getBoundingClientRect().width >= grid.clientWidth * 0.9)) {
      // the axis continues on the detail page
      setTile(max);
      location.href = tile.getAttribute("data-lb-href") || tile.href;
      return;
    }
    flip(function () {
      var before = tile ? tile.getBoundingClientRect().top : 0;
      setTile(Math.min(max, Math.max(TILE_MIN, cur * factor)));
      // keep the anchor tile where it is while the grid reflows
      if (tile) window.scrollBy(0, tile.getBoundingClientRect().top - before);
    });
  }

  function firstVisibleTile() {
    return tiles().filter(function (t) { return t.getBoundingClientRect().bottom > 0; })[0] || tiles()[0] || null;
  }

  document.addEventListener("wheel", function (ev) {
    if (hero || !(ev.ctrlKey || ev.metaKey) || !grids().length) return;
    ev.preventDefault();
    gridZoom(Math.exp(-ev.deltaY * 0.002), ev.target.closest("[data-lb-src]"));
  }, { passive: false });

  document.addEventListener("click", function (ev) {
    var b = ev.target.closest("[data-zoom]");
    if (!b) return;
    ev.preventDefault();
    var zoomIn = b.getAttribute("data-zoom") === "in";
    if (lbZoom) { lbZoom.by(zoomIn ? 1.5 : 1 / 1.5); return; }
    gridZoom(zoomIn ? 1.3 : 1 / 1.3, firstVisibleTile());
  });

})();
