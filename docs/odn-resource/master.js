"use strict";
const OPTS={
  life:["规划","已安装·已测试","在用","已报废"],
  ratio:["1:2","1:4","1:8","1:16","1:32","1:64","1:128"],
  portst:["可用","已使用","已预留","已封锁"],
  lay:["架空","地下","海缆","微沟槽","室内"],
  rowst:["未开始","待处理","已批准","已过期","不适用"],
  pece:["待签署","已签署","已盖章","不适用"],
};
const ENTS=[
  {id:"site",label:"机房管理",pk:"site_code",cols:[
    {k:"site_code",label:"机房编码",req:1,ph:"SITE001"},
    {k:"site_name",label:"机房名称"},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"olt",label:"OLT设备管理",pk:"olt_code",cols:[
    {k:"olt_code",label:"OLT设备编号",req:1,ph:"SITE001_OLT001"},
    {k:"site_code",label:"所属机房",ref:"site",req:1},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"odf",label:"ODF管理",pk:"odf_code",cols:[
    {k:"odf_code",label:"ODF编号",req:1,ph:"ODF001 或文本引用"},
    {k:"site_code",label:"所属机房",ref:"site"},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"occ",label:"OCC管理",pk:"occ_code",cols:[
    {k:"occ_code",label:"OCC编号",req:1},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"odb",label:"ODB管理",pk:"odb_code",cols:[
    {k:"odb_code",label:"ODB编号",req:1},
    {k:"occ_code",label:"上级OCC",ref:"occ",req:1},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"obd",label:"OBD管理",pk:"obd_code",cols:[
    {k:"obd_code",label:"OBD编号",req:1},
    {k:"odb_code",label:"上级ODB",ref:"odb",req:1},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"sdb",label:"SDB管理（一级分光）",pk:"sdb_code",cols:[
    {k:"sdb_code",label:"SDB编号",req:1},
    {k:"odb_code",label:"上级ODB",ref:"odb",req:1},
    {k:"split1_ratio",label:"一级分光比",opt:"ratio",req:1},
    {k:"split1_port",label:"一级分光端口"},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
  {id:"sbd",label:"SBD管理（二级分光）",pk:"sbd_code",cols:[
    {k:"sbd_code",label:"SBD编号",req:1},
    {k:"sdb_code",label:"上级SDB",ref:"sdb",req:1},
    {k:"split2_ratio",label:"二级分光比",opt:"ratio",req:1},
    {k:"split2_port",label:"二级分光端口"},
    {k:"lifecycle_status",label:"资源状态",opt:"life"},
    {k:"remark",label:"备注"}]},
];
const KEY="odn-master-v2";
const panel=document.getElementById("panel");
function esc(s){return String(s==null?"":s).replace(/[&<>"]/g,m=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[m]));}
function emptyDb(){const d={chains:[]};for(const e of ENTS)d[e.id]=[];return d;}
function loadDb(){try{const raw=JSON.parse(localStorage.getItem(KEY)||"{}");const d=emptyDb();
  for(const e of ENTS)if(Array.isArray(raw[e.id]))d[e.id]=raw[e.id];
  if(Array.isArray(raw.chains))d.chains=raw.chains;return d;}catch{return emptyDb();}}
let DB=loadDb();
function saveDb(){try{localStorage.setItem(KEY,JSON.stringify(DB));}catch{}}
const ent=id=>ENTS.find(e=>e.id===id);
const refOptions=id=>DB[id].map(r=>r[ent(id).pk]);
let active="site";
function setMsg(text,cls){const m=document.querySelector(".msg");if(m){m.textContent=text||"";m.className="msg "+(cls||"");}}
function tabBtn(e){const b=document.createElement("button");b.id="tab-"+e.id;
  b.textContent=e.label;if(e.cls)b.className=e.cls;if(active===e.id)b.classList.add("on");
  b.onclick=()=>{active=e.id;buildTabs();renderPanel();};return b;}
function buildTabs(){const nav=document.getElementById("tabs");nav.innerHTML="";
  for(const e of ENTS)nav.append(tabBtn(e));
  nav.append(tabBtn({id:"chains",label:"资源链组装",cls:"chains"}));}
function renderPanel(){panel.innerHTML="";
  if(active==="chains"&&typeof renderChainsPanel==="function")return renderChainsPanel(panel);
  renderEntity(panel,active);}
function ctlHtml(c,cur){cur=cur||"";
  if(c.ref){const opts=refOptions(c.ref);
    return '<select><option value="">'+(opts.length?"（未选）":"（先登记上级）")+"</option>"
      +opts.map(o=>'<option'+(o===cur?" selected":"")+' value="'+esc(o)+'">'+esc(o)+"</option>").join("")+"</select>";}
  if(c.opt)return '<select><option value="">留空</option>'
    +OPTS[c.opt].map(o=>'<option'+(o===cur?" selected":"")+' value="'+esc(o)+'">'+esc(o)+"</option>").join("")+"</select>";
  return '<input value="'+esc(cur)+'" placeholder="'+esc(c.ph||"")+'">';}
function renderEntity(p,id){const e=ent(id),rows=DB[id];
  const tb=document.createElement("div");tb.className="toolbar";
  tb.innerHTML='<span class="hint">共 '+rows.length+' 条；编码登记后不可改，删除时自动检查下级与链路引用</span><span class="msg"></span>';
  p.append(tb);
  const f=document.createElement("div");f.className="frow";
  f.innerHTML="<b>新增</b> "+e.cols.map(c=>"<label>"+c.label+(c.req?" *":"")+" "+ctlHtml(c)+"</label>").join(" ")
    +' <button class="act add">添加</button>';
  p.append(f);
  const wrap=document.createElement("div");wrap.className="gridwrap";
  let h='<table class="grid"><thead><tr><th>#</th>';
  for(const c of e.cols)h+="<th"+(c.ref||c.opt?' class="g2"':"")+">"+c.label+"</th>";
  h+="<th>操作</th></tr></thead><tbody>";
  rows.forEach((r,i)=>{h+='<tr data-i="'+i+'"><td class="pk">'+(i+1)+"</td>";
    for(const c of e.cols)h+=c.k===e.pk?'<td class="pk">'+esc(r[c.k])+"</td>":'<td data-k="'+c.k+'">'+ctlHtml(c,r[c.k])+"</td>";
    h+='<td><button class="act del">删除</button></td></tr>';});
  wrap.innerHTML=h+"</tbody></table>";p.append(wrap);}
function usageOf(id,code){const e=ent(id),out=[];
  for(const o of ENTS)for(const c of o.cols)if(c.ref===id)
    DB[o.id].forEach((r,i)=>{if(r[c.k]===code)out.push(o.label+" 第"+(i+1)+"行");});
  DB.chains.forEach((c,i)=>{if(c[e.pk]===code)out.push("资源链 第"+(i+1)+"条");});
  return out;}
function onPanelChange(ev){const td=ev.target.closest("td[data-k]");if(!td||active==="chains")return;
  const tr=td.closest("tr[data-i]"),row=DB[active][+tr.dataset.i];if(!row)return;
  row[td.dataset.k]=ev.target.value;saveDb();}
function onPanelClick(ev){const b=ev.target;
  if(b.classList.contains("add"))return addFromForm(b.parentElement);
  if(b.classList.contains("del"))return delRow(b.closest("tr[data-i]"));}
function addFromForm(f){const e=ent(active),row={};let miss=[];
  for(const c of e.cols){const ctl=f.querySelector('[data-c="'+c.k+'"]');const v=(ctl.value||"").trim();
    if(c.req&&!v)miss.push(c.label);row[c.k]=v;}
  if(miss.length)return setMsg("必填未填："+miss.join("、"),"err");
  if(DB[active].some(r=>r[e.pk]===row[e.pk]))return setMsg("编码已存在："+row[e.pk],"err");
  DB[active].push(row);saveDb();renderPanel();}
function delRow(tr){if(!tr)return;const i=+tr.dataset.i,e=ent(active),code=DB[active][i][e.pk];
  const used=usageOf(active,code);
  if(used.length)return setMsg("无法删除 "+code+"：被引用于 "+used.slice(0,4).join("、")+(used.length>4?" 等":""),"err");
  if(!confirm("删除 "+e.label+" 里的 "+code+"？"))return;
  DB[active].splice(i,1);saveDb();renderPanel();}
panel.addEventListener("change",onPanelChange);
panel.addEventListener("click",onPanelClick);
buildTabs();
renderPanel();
