/* Búsqueda del sitio: overlay accesible, índice cargado bajo demanda,
   filtrado insensible a acentos y mayúsculas. Sin dependencias. */
(function () {
  "use strict";

  var dialog = document.getElementById("search-dialog");
  var openBtn = document.getElementById("search-open");
  var input = document.getElementById("search-input");
  var results = document.getElementById("search-results");
  if (!dialog || !openBtn || !input || !results) return;

  var index = null; // se carga la primera vez que se abre

  function normalize(s) {
    return s
      .toLowerCase()
      .normalize("NFD")
      .replace(/[\u0300-\u036f]/g, "");
  }

  function loadIndex() {
    if (index) return Promise.resolve(index);
    return fetch("/search-index.json")
      .then(function (r) { return r.json(); })
      .then(function (data) {
        index = data.map(function (e) {
          return {
            entry: e,
            ntitle: normalize(e.title),
            ntext: normalize(e.text || ""),
            npage: normalize(e.page)
          };
        });
        return index;
      });
  }

  function escapeHTML(s) {
    return s.replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }

  // Resalta el término (comparando sin acentos) dentro del texto original.
  function highlight(text, nquery) {
    var ntext = normalize(text);
    var i = ntext.indexOf(nquery);
    if (i < 0) return escapeHTML(text);
    return (
      escapeHTML(text.slice(0, i)) +
      "<mark>" + escapeHTML(text.slice(i, i + nquery.length)) + "</mark>" +
      escapeHTML(text.slice(i + nquery.length))
    );
  }

  // Fragmento de contexto alrededor de la primera coincidencia.
  function snippet(text, nquery) {
    var ntext = normalize(text);
    var i = ntext.indexOf(nquery);
    if (i < 0) return escapeHTML(text.slice(0, 140));
    var start = Math.max(0, i - 50);
    var end = Math.min(text.length, i + nquery.length + 90);
    var frag = (start > 0 ? "…" : "") + text.slice(start, end) + (end < text.length ? "…" : "");
    return highlight(frag, nquery);
  }

  function render(query) {
    var nquery = normalize(query.trim());
    results.innerHTML = "";
    if (nquery.length < 2) return;

    var matches = index
      .map(function (item) {
        var score = 0;
        if (item.ntitle.indexOf(nquery) >= 0) score += 10;
        if (item.npage.indexOf(nquery) >= 0) score += 2;
        if (item.ntext.indexOf(nquery) >= 0) score += 5;
        return { item: item, score: score };
      })
      .filter(function (m) { return m.score > 0; })
      .sort(function (a, b) { return b.score - a.score; })
      .slice(0, 12);

    if (!matches.length) {
      results.innerHTML =
        '<li class="sr-empty" role="status">Sin resultados para «' + escapeHTML(query) + "»</li>";
      return;
    }

    matches.forEach(function (m) {
      var e = m.item.entry;
      var li = document.createElement("li");
      li.innerHTML =
        '<a href="' + e.url + '">' +
        '<span class="sr-page">' + escapeHTML(e.page) + "</span>" +
        '<span class="sr-title">' + highlight(e.title, nquery) + "</span> " +
        '<span class="sr-text">' + snippet(e.text || "", nquery) + "</span>" +
        "</a>";
      results.appendChild(li);
    });
  }

  function open() {
    loadIndex().then(function () {
      if (!dialog.open) dialog.showModal();
      input.focus();
      input.select();
    });
  }

  openBtn.addEventListener("click", open);

  document.addEventListener("keydown", function (ev) {
    var typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
    if ((ev.key === "/" && !typing) || (ev.key.toLowerCase() === "k" && (ev.ctrlKey || ev.metaKey))) {
      ev.preventDefault();
      open();
    }
  });

  input.addEventListener("input", function () { render(input.value); });

  // Enter navega al primer resultado. Esc cierra siempre con una sola
  // pulsación (los input[type=search] de Chrome consumen el primer Esc
  // para limpiar el texto).
  input.addEventListener("keydown", function (ev) {
    if (ev.key === "Enter") {
      ev.preventDefault();
      var first = results.querySelector("a");
      if (first) {
        dialog.close();
        window.location.href = first.getAttribute("href");
      }
    } else if (ev.key === "Escape") {
      ev.preventDefault();
      dialog.close();
    }
  });

  // Al navegar dentro de la misma página (anclas), cerrar el diálogo.
  results.addEventListener("click", function (ev) {
    if (ev.target.closest("a")) dialog.close();
  });
})();
