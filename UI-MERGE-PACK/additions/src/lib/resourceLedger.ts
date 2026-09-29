import type { ResourceDimension } from '../types/truth';
export function modelAAvailable(total:number, ownerReserve:number, reserved:number, allocated:number):number { return total-ownerReserve-reserved-allocated; }
export function verifyModelA(d: ResourceDimension): boolean | null {
  const xs=[d.total,d.ownerReserve,d.reserved,d.allocated,d.available];
  if (xs.some(x=>x.value===null || x.state==='UNKNOWN' || x.state==='UNAVAILABLE')) return null;
  return d.available.value === modelAAvailable(d.total.value!,d.ownerReserve.value!,d.reserved.value!,d.allocated.value!);
}
