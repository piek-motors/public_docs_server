package main

const htmlTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
        </div>

        <div class="search-container">
            <form class="search-form" id="searchForm">
                <input type="text" class="search-input" id="searchInput" placeholder="Поиск по ID (например: ВЗИС.421321.028)" required>
                <button type="submit" class="search-button">Найти</button>
            </form>
            <div class="search-results" id="searchResults"></div>
        </div>

        <div class="breadcrumb">
            {{range $i, $part := .Breadcrumb}}
                {{if $i}} / {{end}}
                <a href="/browse{{$part.Path}}">{{$part.Name}}</a>
            {{end}}
        </div>
        
        <div class="content">
            {{if .Data.Directories}}
            <div class="section" id="directories-section">
                <h2>Папки</h2>
                <ul class="file-list">
                    {{range .Data.Directories}}
                    <li class="file-item directory">
                        <a href="/browse/{{.RelativePath}}" class="file-link">
                            <div class="file-info">
                                <div class="file-name">📁 {{.Name}}</div>
                            </div>
                        </a>
                    </li>
                    {{end}}
                </ul>
            </div>
            {{end}}

            {{if .Data.Files}}
            <div class="section" id="files-section">
                <h2>Документы</h2>
                <ul class="file-list">
                    {{range .Data.Files}}
                     <li class="file-item">
                        <a href="/browse/{{.RelativePath}}" class="file-link" target="_blank">
                            <div class="file-info">
                                <div class="file-name">{{.Name}}</div>
                                <div class="file-details">
                                    <span>{{.ModTime.Format "02.01.2006"}}</span>
                                </div>
                            </div>
                        </a>
                    </li>
                    {{end}}
                </ul>
            </div>
            {{else}}
                <div class="section">
                     <div class="empty-state" id="no-files">
                        <div class="icon">📄</div>
                        <p>Нет документов в этом разделе</p>
                    </div>
                </div>
            {{end}}

            {{if and (not .Data.Directories) (not .Data.Files)}}
            <div class="section">
                <div class="empty-state" id="empty-directory">
                    <div class="icon">📁</div>
                    <p>Этот раздел пуст</p>
                </div>
            </div>
            {{end}}
        </div>
    </div>
    <script src="/static/js/main.js"></script>
</body>
</html>` 