"use strict";
const CKEYS=["lifecycle_status","site_code","site_name","olt_code","odf_code","odf_port","occ_code","odb_code","obd_code","split1_ratio","split1_port","sdb_code","sbd_code","split2_ratio","split2_port","total_split","fiber_code","fr_to","port_status","laying_method","row_status","pece_status","remark"];
const CLABELS=["资源状态","机房编码","机房名称","OLT设备编号","ODF编号","ODF端口","OCC编号","ODB编号","OBD编号","一级分光比","一级分光端口","SDB编号","SBD编号","二级分光比","二级分光端口","总分光比","光缆/纤芯编号","FR/TO标签","端口状态","敷设方式","ROW状态","PECE状态","备注"];
const CFIELDS=[
  {id:"f_site",label:"机房 *",ref:1},{id:"f_olt",label:"OLT设备 *",ref:1},
  {id:"f_odf",label:"ODF编号（可空）",ref:1},{id:"f_odfport",label:"ODF端口（可空）",text:1},
  {id:"f_occ",label:"OCC *",ref:1},{id:"f_odb",label:"ODB *",ref:1},
  {id:"f_obd",label:"OBD（可空）",ref:1},{id:"f_sdb",label:"SDB（一级分光，可空）",ref:1},
  {id:"f_sbd",label:"SBD（二级分光，可空）",ref:1},
  {id:"f_status",label:"资源状态（留空=规划）",opt:"life"},{id:"f_portst",label:"端口状态",opt:"portst"},
  {id:"f_lay",label:"敷设方式",opt:"lay"},{id:"f_rowst",label:"ROW状态",opt:"rowst"},{id:"f_pece",label:"PECE状态",opt:"pece"},
  {id:"f_fiber",label:"光缆/纤芯编号",text:1},{id:"f_frto",label:"FR/TO标签",text:1},{id:"f_remark",label:"备注",text:1},
];
const V=id=>document.getElementById(id).value;
function ratioDen(r){const m=/^1\s*:\s*(\d+)$/.exec((r||"").trim());return m?+m[1]:0;}
function totalSplit(a,b){const x=ratioDen(a),y=ratioDen(b);
  if(!x&&!y)return"";if(!y)return"1:"+x;if(!x)return"1:"+y;return"1:"+(x*y);}
async function fp(row){const s=CKEYS.map(k=>(row[k]==null?"":String(row[k])).trim()).join("\x1f");
  try{const b=await crypto.subtle.digest("SHA-256",new TextEncoder().encode(s));
    return[...new Uint8Array(b)].map(x=>x.toString(16).padStart(2,"0")).join("");}
  catch{let h=2166136261;for(const ch of s){h^=ch.codePointAt(0);h=Math.imul(h,16777619)>>>0;}return"fnv"+h.toString(16);}}
function composerEl(){const d=document.createElement("div");d.className="composer";
  for(const f of CFIELDS){const fld=document.createElement("div");fld.className="fld";
    const lab=document.createElement("label");lab.textContent=f.label;lab.htmlFor=f.id;
    const ctl=document.createElement(f.text?"input":"select");ctl.id=f.id;
    if(f.opt)ctl.innerHTML='<option value="">留空</option>'+OPTS[f.opt].map(o=>'<option value="'+o+'">'+o+"</option>").join("");
    fld.append(lab,ctl);d.append(fld);}
  const dv=document.createElement("div");dv.className="derived";
  dv.innerHTML='<span>一级分光比 <b id="d_sp1">—</b></span><span>一级分光端口 <b id="d_p1">—</b></span>'
    +'<span>二级分光比 <b id="d_sp2">—</b></span><span>二级分光端口 <b id="d_p2">—</b></span>'
    +'<span>总分光比（自动） <b id="d_total">—</b></span>';
  d.append(dv);
  const bar=document.createElement("div");bar.className="fld";bar.style.gridColumn="1/-1";
  bar.innerHTML='<button class="act" id="gen">生成链路</button><span class="msg"></span>';
  d.append(bar);return d;}
function fillSel(id,codes,hint){const s=document.getElementById(id),cur=s.value;
  s.innerHTML='<option value="">'+hint+"</option>"+codes.map(c=>'<option value="'+esc(c)+'">'+esc(c)+"</option>").join("");
  if(codes.includes(cur))s.value=cur;}
function refreshComposer(){const siteV=V("f_site"),occV=V("f_occ"),odbV=V("f_odb"),sdbV=V("f_sdb");
  fillSel("f_site",DB.site.map(r=>r.site_code),"（先在机房管理登记）");
  fillSel("f_occ",DB.occ.map(r=>r.occ_code),"（先在OCC管理登记）");
  fillSel("f_olt",DB.olt.filter(r=>!siteV||r.site_code===siteV).map(r=>r.olt_code),siteV?"（无，先登记OLT）":"（可先选机房过滤）");
  fillSel("f_odf",DB.odf.filter(r=>!siteV||r.site_code===siteV).map(r=>r.odf_code),"（可空）");
  fillSel("f_odb",DB.odb.filter(r=>!occV||r.occ_code===occV).map(r=>r.odb_code),occV?"（无，先登记ODB）":"（先选OCC）");
  fillSel("f_obd",DB.obd.filter(r=>!odbV||r.odb_code===odbV).map(r=>r.obd_code),"（可空，仅登记到ODB）");
  fillSel("f_sdb",DB.sdb.filter(r=>!odbV||r.odb_code===odbV).map(r=>r.sdb_code),"（可空，无一级分光）");
  fillSel("f_sbd",DB.sbd.filter(r=>!sdbV||r.sdb_code===sdbV).map(r=>r.sbd_code),"（可空，无二级分光）");
  updateDerived();}
function updateDerived(){const sdb=DB.sdb.find(r=>r.sdb_code===V("f_sdb")),sbd=DB.sbd.find(r=>r.sbd_code===V("f_sbd"));
  const g=(o,k)=>o?(o[k]||"（未填）"):"—";
  document.getElementById("d_sp1").textContent=g(sdb,"split1_ratio");
  document.getElementById("d_p1").textContent=g(sdb,"split1_port");
  document.getElementById("d_sp2").textContent=g(sbd,"split2_ratio");
  document.getElementById("d_p2").textContent=g(sbd,"split2_port");
  document.getElementById("d_total").textContent=sdb||sbd
    ?totalSplit(sdb&&sdb.split1_ratio,sbd&&sbd.split2_ratio)||"（分光比未填全）":"—";}
async function genChain(){const names={f_site:"机房",f_olt:"OLT",f_occ:"OCC",f_odb:"ODB"};
  const miss=Object.keys(names).filter(id=>!V(id));
  if(miss.length)return setMsg("必选未选："+miss.map(id=>names[id]).join("、"),"err");
  const sdb=DB.sdb.find(r=>r.sdb_code===V("f_sdb")),sbd=DB.sbd.find(r=>r.sbd_code===V("f_sbd"));
  if(sbd&&!sdb)return setMsg("登记 SBD 前须先选 SDB","err");
  if(sdb&&!V("f_obd"))return setMsg("登记 SDB 前须先选 OBD","err");
  const site=DB.site.find(r=>r.site_code===V("f_site"))||{};
  const row={lifecycle_status:V("f_status"),site_code:V("f_site"),site_name:site.site_name||"",
    olt_code:V("f_olt"),odf_code:V("f_odf"),odf_port:(V("f_odfport")||"").trim(),
    occ_code:V("f_occ"),odb_code:V("f_odb"),obd_code:V("f_obd"),
    sdb_code:sdb?sdb.sdb_code:"",sbd_code:sbd?sbd.sbd_code:"",
    split1_ratio:sdb?(sdb.split1_ratio||""):"",split1_port:sdb?(sdb.split1_port||""):"",
    split2_ratio:sbd?(sbd.split2_ratio||""):"",split2_port:sbd?(sbd.split2_port||""):"",
    fiber_code:(V("f_fiber")||"").trim(),fr_to:(V("f_frto")||"").trim(),
    port_status:V("f_portst"),laying_method:V("f_lay"),row_status:V("f_rowst"),pece_status:V("f_pece"),
    remark:(V("f_remark")||"").trim()};
  row.total_split=totalSplit(row.split1_ratio,row.split2_ratio);
  const print=await fp(row);
  for(let i=0;i<DB.chains.length;i++){if(await fp(DB.chains[i])===print)
    return setMsg("与第 "+(i+1)+" 条链路指纹重复，导入会被拒绝，未生成","err");}
  DB.chains.push(row);saveDb();rerenderList();
  setMsg("已生成第 "+DB.chains.length+" 条链路，总分光比 "+(row.total_split||"（无分光）"),"ok");}
function chainListEl(){const d=document.createElement("div");d.id="chainlist";
  const tb=document.createElement("div");tb.className="toolbar";
  tb.innerHTML='<span class="hint">链路共 '+DB.chains.length+' 条；导出为 23 列模板格式（表头+数据行）</span>'
    +' <button class="act" id="csv">导出 CSV</button>'
    +(DB.chains.length?"":'<span class="msg ok">还没有链路：先在各管理页登记基础数据，再在上方逐级选择生成</span>');
  d.append(tb);
  if(!DB.chains.length)return d;
  const wrap=document.createElement("div");wrap.className="gridwrap";
  let h='<table class="grid"><thead><tr><th>#</th>';
  for(const l of CLABELS)h+="<th>"+l+"</th>";
  h+="<th>操作</th></tr></thead><tbody>";
  DB.chains.forEach((c,i)=>{h+='<tr data-i="'+i+'"><td class="pk">'+(i+1)+"</td>";
    for(const k of CKEYS)h+='<td'+(k==="total_split"?' class="auto"':"")+">"+esc(c[k])+"</td>";
    h+='<td><button class="act cdel">删除</button></td></tr>';});
  wrap.innerHTML=h+"</tbody></table>";d.append(wrap);return d;}
function rerenderList(){const old=document.getElementById("chainlist");
  if(old)old.replaceWith(chainListEl());
  const csv=document.getElementById("csv");if(csv)csv.onclick=exportCsv;}
function exportCsv(){const q=v=>{v=v==null?"":String(v).trim();
    return /[",\n]/.test(v)?'"'+v.replace(/"/g,'""')+'"':v;};
  const lines=[CLABELS.map(q).join(",")];
  for(const c of DB.chains)lines.push(CKEYS.map(k=>q(c[k])).join(","));
  const blob=new Blob(["\ufeff"+lines.join("\r\n")],{type:"text/csv;charset=utf-8"});
  const a=document.createElement("a");a.href=URL.createObjectURL(blob);
  a.download="ODN资源链_导出_"+new Date().toISOString().slice(0,10)+".csv";a.click();
  URL.revokeObjectURL(a.href);}
function renderChainsPanel(p){p.append(composerEl());p.append(chainListEl());
  for(const s of p.querySelectorAll(".composer select"))s.addEventListener("change",refreshComposer);
  document.getElementById("gen").onclick=genChain;
  const csv=document.getElementById("csv");if(csv)csv.onclick=exportCsv;
  refreshComposer();}
panel.addEventListener("click",ev=>{const b=ev.target.closest("button.cdel");if(!b)return;
  const i=+b.closest("tr[data-i]").dataset.i;
  if(!confirm("删除链路第 "+(i+1)+" 条？"))return;
  DB.chains.splice(i,1);saveDb();rerenderList();});
