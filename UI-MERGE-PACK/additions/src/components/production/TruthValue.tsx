import React from 'react'; import type { TruthEnvelope } from '../../types/truth';
export function TruthValue<T>({label,envelope,format=(v)=>String(v)}:{label:string;envelope:TruthEnvelope<T>;format?:(v:T)=>string}) {
 const missing=envelope.value===null; const value=missing?(envelope.state==='UNAVAILABLE'?'Unavailable':'Unknown'):format(envelope.value as T);
 return <div className="truth-value" data-state={envelope.state} data-freshness={envelope.freshness}>
  <div className="truth-value__label">{label}</div><div className="truth-value__value">{value}</div>
  <div className="truth-value__meta"><span>{envelope.state}</span><span>{envelope.freshness}</span>{envelope.source&&<span>{envelope.source}</span>}{envelope.observedAt&&<time dateTime={envelope.observedAt}>{envelope.observedAt}</time>}</div>
 </div>;
}
