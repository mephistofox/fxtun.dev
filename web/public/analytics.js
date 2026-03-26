// Google Analytics 4 (domain-specific tracking IDs)
(function(){
  var ids = { 'fxtun.ru': 'G-TKFQMYKJZZ', 'fxtun.dev': 'G-4FH5VTH49H' };
  var id = ids[location.hostname];
  if (!id) return;
  var s = document.createElement('script');
  s.async = true;
  s.src = 'https://www.googletagmanager.com/gtag/js?id=' + id;
  document.head.appendChild(s);
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  window.gtag = gtag;
  gtag('js', new Date());
  gtag('config', id, { send_page_view: false });
})();

// Yandex.Metrika (fxtun.ru only)
(function(){
  if (location.hostname !== 'fxtun.ru') return;
  (function(m,e,t,r,i,k,a){
    m[i]=m[i]||function(){(m[i].a=m[i].a||[]).push(arguments)};
    m[i].l=1*new Date();
    for(var j=0;j<document.scripts.length;j++){if(document.scripts[j].src===r)return;}
    k=e.createElement(t),a=e.getElementsByTagName(t)[0],k.async=1,k.src=r,a.parentNode.insertBefore(k,a)
  })(window,document,'script','https://mc.yandex.ru/metrika/tag.js?id=108256538','ym');
  ym(108256538,'init',{ssr:true,webvisor:true,clickmap:true,ecommerce:"dataLayer",referrer:document.referrer,url:location.href,accurateTrackBounce:true,trackLinks:true});
})();
