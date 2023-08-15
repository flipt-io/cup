import Image from 'next/image';
import logo from './public/images/cup.svg';

export default {
  logo: (
    <>
      <Image src={logo} alt="Cup - Git Contribution Automation" width={45} />
    </>
  ),
  project: {
    link: 'https://github.com/flipt-io/cup'
  }
  // ... other theme options
}
