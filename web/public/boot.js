// Runs before first paint so the saved theme and language never flash.
(function () {
  var root = document.documentElement;
  var lang = 'fa';
  var theme = 'system';
  try {
    lang = localStorage.getItem('karvan.lang') === 'en' ? 'en' : 'fa';
    var saved = localStorage.getItem('karvan.theme');
    if (saved === 'light' || saved === 'dark') theme = saved;
  } catch (e) {
    // Storage can be blocked; the defaults above still apply.
  }
  if (theme === 'system') {
    theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  root.lang = lang;
  root.dir = lang === 'fa' ? 'rtl' : 'ltr';
  root.dataset.mode = theme;
  root.dataset.theme = 'karvan';
})();
