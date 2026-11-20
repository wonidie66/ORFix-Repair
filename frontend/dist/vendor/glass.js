/*
 * Based on react-liquid-glass-svg 1.0.5 (MIT)
 * Copyright (c) 2026 Yurii Khvyshchuk
 * https://github.com/yurkagon/react-liquid-glass-svg
 * UMD/React.createElement port, minimal filter implementation.
 * This notice and LICENSE_Glass_MIT.txt must be retained.
 */
(function(){
  var h=React.createElement, num=0;
  function Glass(props){
    var id='orf-liquid-'+(++num);
    var br=props.radius||24, blur=props.blur==null?8:props.blur;
    var filter=h('svg',{style:{position:'absolute',width:0,height:0,pointerEvents:'none'},'aria-hidden':'true'},
      h('defs',null,h('filter',{id:id,x:'-10%',y:'-10%',width:'120%',height:'120%'},
        h('feTurbulence',{type:'fractalNoise',baseFrequency:'0.008',numOctaves:'1',seed:'2',result:'turbulence'}),
        h('feGaussianBlur',{in:'turbulence',stdDeviation:'3',result:'softMap'}),
        h('feDisplacementMap',{in:'SourceGraphic',in2:'softMap',scale:'26',xChannelSelector:'R',yChannelSelector:'G'})
      )));
    return h('div',{className:'liquid-frame '+(props.className||''),style:Object.assign({borderRadius:br},props.style||{})},
      filter,
      h('div',{className:'liquid-optics',style:{borderRadius:br,backdropFilter:'blur('+blur+'px) saturate(1.1)',WebkitBackdropFilter:'blur('+blur+'px) saturate(1.1)',filter:'url(#'+id+')'}}),
      h('div',{className:'liquid-tint',style:{borderRadius:br}}),
      h('div',{className:'liquid-content'},props.children));
  }
  window.ORFixLiquidGlass=Glass;
})();