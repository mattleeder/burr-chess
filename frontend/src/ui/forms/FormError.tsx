export function FormError({ errorMessage }: { errorMessage: string }) {
  let className = "formError"
  if (errorMessage == "") {
    className += " hidden"
  }
  return (
    <span className={className}>{errorMessage}</span>
  )
}