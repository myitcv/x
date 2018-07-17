package main

const homePageTemplate = `
				{{ $pkg := .Pkg }}
				<!DOCTYPE html>
				<html>
					<head>
						<meta charset="UTF-8">
						<title>gitgodoc server for {{.Pkg}}</title>
						<link rel="stylesheet" href="/__static/bootstrap.min.css" />
						<style>
							body {
								text-align:center;
							}
							.list-group-item {
								font-size:1.3em;
							}
							.branch-name {
								font-weight:600;
							}
						</style>
					</head>
					<body>
						<h1><b>gitgodoc</b></h1>
						<h3><code>{{.Pkg}}</code></h3>
						<hr/>
						<div class="container">
							<div class="row">
								<div class="col-md-12">
									<div class="panel default-panel">
										<div class="panel-heading"><h3><b>Branches</b><h3></div>
											{{with .Branches}}
											<div class="panel-body">
												<ul class="list-group">
												{{ range . }}{{ if eq . "master" }}<li class="list-group-item list-group-item-info"><a class="branch-name" href="/{{.}}">{{ . }}</a> (<a href="/{{.}}/pkg/{{$pkg}}/">{{$pkg}}</a>)</li>{{ end }}{{ end }}
												</ul>
												<ul class="list-group">
												{{ range . }}{{ if ne . "master" }}<li class="list-group-item"><a class="branch-name" href="/{{.}}">{{ . }}</a> (<a href="/{{.}}/pkg/{{$pkg}}/">{{$pkg}}</a>)</li>{{ end }}{{ end }}
												</ul>
											{{else}}
											<div><h4><b>No branches known to godoc server</b></h4></div>
											{{end}}
											</div>
										</div>
									<div>
								</div>
							</div>
					</body>
				</html>`
