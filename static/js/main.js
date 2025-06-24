document.getElementById('searchForm').addEventListener('submit', async function (e) {
  e.preventDefault();
  const query = document.getElementById('searchInput').value.trim();
  if (!query) return;
  const resultsDiv = document.getElementById('searchResults');
  resultsDiv.innerHTML = '<p>Поиск...</p>';
  try {
    const response = await fetch('/api/search?id=' + encodeURIComponent(query));
    const data = await response.json();
    if (data.count === 0) {
      resultsDiv.innerHTML = '<p>Документы с ID "' + query + '" не найдены</p>';
      return;
    }
    let html = '<h4>Найдено документов: ' + data.count + '</h4>';
    data.results.forEach(function (doc) {
      html += '<div class="search-result-item">';
      html += '<h3><a href="/browse/' + doc.path + '" class="search-result-link" target="_blank">' + doc.name + '</a></h3>';
      html += '<p><strong>Путь:</strong> ' + doc.path + '</p>';
      html += '<p><strong>Дата изменения:</strong> ' + new Date(doc.mod_time).toLocaleDateString('ru-RU') + '</p>';
      html += '</div>';
    });
    resultsDiv.innerHTML = html;
  } catch (error) {
    resultsDiv.innerHTML = '<p>Ошибка при поиске: ' + error.message + '</p>';
  }
});

function formatFileSize(bytes) {
  if (bytes === 0) return '0 Б';
  const k = 1024;
  const sizes = ['Б', 'КБ', 'МБ', 'ГБ'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
} 