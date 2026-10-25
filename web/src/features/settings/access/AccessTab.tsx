import { AddressCard } from './AddressCard';
import { LoginCard } from './LoginCard';

/** The panel address (hidden path and port) and the operator login. */
export function AccessTab() {
  return (
    <>
      <AddressCard />
      <LoginCard />
    </>
  );
}
