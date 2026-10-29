package chartframe

import (
	"bytes"
	"errors"
	"html/template"
)

func htmlDocument(title string, svg []byte) ([]byte, error) {
	i := bytes.Index(svg, []byte("<svg"))
	if i < 0 {
		return nil, errors.New("chartframe: SVG root missing")
	}
	var b bytes.Buffer
	// The SVG is generated exclusively by our validated Gonum adapter. Labels
	// have already been XML-escaped; never pass external SVG to this helper.
	err := pageTemplate.Execute(&b, struct {
		Title string
		SVG   template.HTML
	}{title, template.HTML(svg[i:])})
	return b.Bytes(), err
}

var pageTemplate = template.Must(template.New("chart").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}} · chartframe</title><style>
*{box-sizing:border-box}body{margin:0;background:#f2f5f7;color:#142b3d;font:15px system-ui,sans-serif}header,main,footer{max-width:1440px;margin:auto;padding:24px}header{display:flex;gap:16px;align-items:center;justify-content:space-between;flex-wrap:wrap}h1{margin:8px 0;font-size:clamp(20px,4vw,30px)}small{color:#42706c;font-weight:700;letter-spacing:.08em}button{border:1px solid #c8d5de;background:white;color:#142b3d;border-radius:8px;padding:10px 14px;cursor:pointer;font:inherit}button:hover{background:#e0f2ed}button:focus-visible{outline:3px solid #00a58d}nav{display:flex;gap:6px;flex-wrap:wrap}.viewport{background:white;border:1px solid #d5dfe5;border-radius:12px;overflow:hidden;box-shadow:0 12px 40px #142b3d0c}.viewport svg{width:100%;height:auto;display:block;cursor:grab;touch-action:none}.viewport svg.dragging{cursor:grabbing}main{padding-top:0}p,footer{color:#58707e;line-height:1.7}footer{padding-top:0;font-size:13px}@media print{nav,.hint,footer{display:none}body{background:white}.viewport{border:0;box-shadow:none}}
</style></head><body><header><div><small>CHARTFRAME / GO SCIENCE</small><h1>{{.Title}}</h1></div><nav aria-label="Chart controls"><button id="in" aria-label="Zoom in">＋</button><button id="out" aria-label="Zoom out">−</button><button id="reset">Reset view</button><button id="download">Download SVG</button></nav></header><main><div class="viewport">{{.SVG}}</div><p class="hint">Scroll or use the buttons to zoom. Drag to pan. This changes the viewport; axes and data are not recalculated.</p></main><footer>Generated with chartframe · Native Go, rendered with Gonum Plot · No network connection required.</footer><script>
(()=>{'use strict';const svg=document.querySelector('.viewport svg');const original=svg.getAttribute('viewBox').split(/\s+/).map(Number);let view=original.slice(),start=null;const apply=()=>svg.setAttribute('viewBox',view.join(' '));const zoom=f=>{const nw=view[2]*f,nh=view[3]*f;if(nw<original[2]/20||nw>original[2]*5)return;view=[view[0]+(view[2]-nw)/2,view[1]+(view[3]-nh)/2,nw,nh];apply()};document.getElementById('in').onclick=()=>zoom(.8);document.getElementById('out').onclick=()=>zoom(1.25);document.getElementById('reset').onclick=()=>{view=original.slice();apply()};svg.addEventListener('wheel',e=>{e.preventDefault();zoom(e.deltaY<0?.9:1.1)},{passive:false});svg.addEventListener('pointerdown',e=>{if(e.button!==0)return;start=[e.clientX,e.clientY,...view];svg.setPointerCapture(e.pointerId);svg.classList.add('dragging')});svg.addEventListener('pointermove',e=>{if(!start)return;const r=svg.getBoundingClientRect();view[0]=start[2]-(e.clientX-start[0])*start[4]/r.width;view[1]=start[3]-(e.clientY-start[1])*start[5]/r.height;apply()});const end=()=>{start=null;svg.classList.remove('dragging')};svg.addEventListener('pointerup',end);svg.addEventListener('pointercancel',end);document.getElementById('download').onclick=()=>{const copy=svg.cloneNode(true);copy.setAttribute('viewBox',original.join(' '));const url=URL.createObjectURL(new Blob([copy.outerHTML],{type:'image/svg+xml'}));const a=document.createElement('a');a.href=url;a.download='chart.svg';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)};})();
</script></body></html>`))
