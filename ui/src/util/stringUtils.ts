export function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export function allCaps(str: string): string {
  return str.toUpperCase();
}
