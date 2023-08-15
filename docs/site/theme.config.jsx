import { useRouter } from 'next/router'
import { useConfig } from 'nextra-theme-docs'
import Image from 'next/image';
import logo from './public/images/cup.svg';

export default {
  docsRepositoryBase: 'https://github.com/flipt-io/cup',
  head: () => {
    const { asPath, defaultLocale, locale } = useRouter()
    const { frontMatter } = useConfig()
    const url =
      `https://${ process.env.VERCEL_URL ?? 'localhost:3000'}` +
      (defaultLocale === locale ? asPath : `/${locale}${asPath}`)

    return (
      <>
        <meta property="og:url" content={url} />
        <meta property="og:title" content={frontMatter.title || 'Cup'} />
        <meta
          property="og:description"
          content={frontMatter.description || 'Cup - Contribution automation for Git'}
        />
      </>
    )
  },
  logo: (
    <>
      <Image src={logo} alt="Cup - Git Contribution Automation" width={50} />
      <span style={{ marginLeft: '.4rem', fontWeight: 800 }}>Cup</span>
    </>
  ),
  project: {
    link: 'https://github.com/flipt-io/cup'
  },
  useNextSeoProps: () => {
    return {
      titleTemplate: '%s - Cup'
    }
  },
  primaryHue: 26
  // ... other theme options
}
