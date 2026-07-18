/* Navegación móvil, animaciones de aparición, lightbox y service worker. */
(function () {
  "use strict";

  // --- Menú móvil ---
  var toggle = document.getElementById("nav-toggle");
  var nav = document.getElementById("nav");
  if (toggle && nav) {
    toggle.addEventListener("click", function () {
      var open = nav.classList.toggle("open");
      toggle.setAttribute("aria-expanded", String(open));
      toggle.setAttribute("aria-label", open ? "Cerrar menú" : "Abrir menú");
    });
  }

  // --- Aparición sutil al hacer scroll ---
  var reveals = document.querySelectorAll(".reveal");
  if ("IntersectionObserver" in window && reveals.length) {
    var io = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (e) {
          if (e.isIntersecting) {
            e.target.classList.add("in");
            io.unobserve(e.target);
          }
        });
      },
      { threshold: 0.12 }
    );
    reveals.forEach(function (el) { io.observe(el); });
  } else {
    reveals.forEach(function (el) { el.classList.add("no-observer"); });
  }

  // --- Lightbox de la galería ---
  var lightbox = document.getElementById("lightbox");
  if (lightbox) {
    var img = document.getElementById("lightbox-img");
    var caption = document.getElementById("lightbox-caption");
    document.querySelectorAll(".lightbox-link").forEach(function (link) {
      link.addEventListener("click", function (ev) {
        ev.preventDefault();
        img.src = link.getAttribute("href");
        img.alt = link.dataset.caption || "";
        caption.textContent = link.dataset.caption || "";
        lightbox.showModal();
      });
    });
    lightbox.querySelector(".lightbox-close").addEventListener("click", function () {
      lightbox.close();
    });
    lightbox.addEventListener("click", function (ev) {
      if (ev.target === lightbox) lightbox.close();
    });
  }

  // --- Service worker (PWA) ---
  if ("serviceWorker" in navigator) {
    window.addEventListener("load", function () {
      navigator.serviceWorker.register("/sw.js").catch(function () {
        /* sin conexión o entorno sin soporte: la página funciona igual */
      });
    });
  }
})();
