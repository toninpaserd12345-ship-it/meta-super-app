import{g as e,v as t,y as a}from"../virtual/entry.mjs";function createSimpleFunctional(l,s="div",i){return e()({name:i??t.capitalize(t.camelize(l.replace(/__/g,"-"))),props:{tag:{type:String,default:s},...a()},setup:(e,{slots:a})=>()=>t.h(e.tag,{class:[l,e.class],style:e.style},a.default?.())})}export{createSimpleFunctional as c};
//# sourceMappingURL=createSimpleFunctional-Dn3pQNXQ.mjs.map
