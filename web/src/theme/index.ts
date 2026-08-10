import { extendTheme, defineStyleConfig } from '@chakra-ui/react'

// FastGPT 风格的颜色系统
const colors = {
  // 主色调 - 蓝色系
  primary: {
    50: '#F0F4FF',
    100: '#E1EAFF',
    200: '#C5D7FF',
    300: '#94B5FF',
    400: '#5E8FFF',
    500: '#487FFF',
    600: '#3370FF',
    700: '#2B5FD9',
    800: '#2450B5',
    900: '#1D4091',
  },
  // 灰色系
  myGray: {
    50: '#F6F8FA',
    100: '#F3F4F6',
    150: '#EEF0F3',
    200: '#D0D7DE',
    250: '#BFC8D1',
    300: '#B1BAC4',
    400: '#8B949E',
    500: '#6E7781',
    600: '#57606A',
    700: '#424A53',
    800: '#32383F',
    900: '#24292F',
  },
  // 白色系
  myWhite: {
    100: '#FEFEFE',
    200: '#FDFDFE',
    300: '#FBFBFC',
    400: '#F8FAFB',
    500: '#F6F8F9',
    600: '#F4F6F8',
  },
  // 红色系
  red: {
    50: '#FEF3F2',
    100: '#FEE4E2',
    200: '#FECDCA',
    300: '#FDA29B',
    400: '#F97066',
    500: '#F04438',
    600: '#D92D20',
    700: '#B42318',
    800: '#912018',
    900: '#7A271A',
  },
  // 绿色系
  green: {
    50: '#ECFDF3',
    100: '#D1FADF',
    200: '#A6F4C5',
    300: '#6CE9A6',
    400: '#32D583',
    500: '#12B76A',
    600: '#039855',
    700: '#027A48',
    800: '#05603A',
    900: '#054F31',
  },
  // 黄色系
  yellow: {
    50: '#FFFAEB',
    100: '#FEF0C7',
    200: '#FEDF89',
    300: '#FEC84B',
    400: '#FDB022',
    500: '#F79009',
    600: '#DC6803',
    700: '#B54708',
    800: '#93370D',
    900: '#7A2E0E',
  },
}

// 按钮样式配置
const Button = defineStyleConfig({
  baseStyle: {
    fontWeight: 'medium',
    borderRadius: 'sm',
    _active: {
      transform: 'scale(0.98)',
    },
  },
  sizes: {
    xs: {
      fontSize: 'xs',
      px: 2,
      h: '24px',
    },
    sm: {
      fontSize: 'sm',
      px: 3,
      h: '30px',
    },
    md: {
      fontSize: 'sm',
      px: 4,
      h: '34px',
    },
    lg: {
      fontSize: 'md',
      px: 4,
      h: '40px',
    },
  },
  variants: {
    // 主要按钮 - 蓝色填充
    primary: {
      bg: 'primary.600',
      color: 'white',
      _hover: {
        bg: 'primary.700',
        _disabled: {
          bg: 'primary.600',
        },
      },
    },
    // 主要按钮轮廓
    primaryOutline: {
      color: 'primary.600',
      border: '1px solid',
      borderColor: 'primary.300',
      bg: 'white',
      _hover: {
        bg: 'primary.50',
      },
    },
    // 白色基础按钮
    whiteBase: {
      color: 'myGray.600',
      border: '1px solid',
      borderColor: 'myGray.250',
      bg: 'white',
      _hover: {
        color: 'primary.600',
        borderColor: 'primary.300',
      },
    },
    // 危险按钮
    danger: {
      bg: 'red.600',
      color: 'white',
      _hover: {
        bg: 'red.700',
      },
    },
    // 灰色按钮
    gray: {
      bg: 'myGray.100',
      color: 'myGray.700',
      _hover: {
        bg: 'myGray.150',
      },
    },
  },
  defaultProps: {
    variant: 'primary',
    size: 'md',
  },
})

// 输入框样式配置
const Input = defineStyleConfig({
  baseStyle: {
    field: {
      borderRadius: 'sm',
      bg: 'white',
    },
  },
  variants: {
    outline: {
      field: {
        bg: 'white',
        border: '1px solid',
        borderColor: 'myGray.200',
        _focus: {
          borderColor: 'primary.500',
          boxShadow: '0px 0px 0px 2.4px rgba(51, 112, 255, 0.15)',
        },
        _hover: {
          borderColor: 'primary.300',
        },
      },
    },
  },
  defaultProps: {
    variant: 'outline',
  },
})

// 卡片样式
const Card = defineStyleConfig({
  baseStyle: {
    container: {
      bg: 'white',
      borderRadius: 'lg',
      border: '1px solid',
      borderColor: 'myGray.200',
      boxShadow: '0px 0px 1px 0px rgba(19, 51, 107, 0.08), 0px 1px 2px 0px rgba(19, 51, 107, 0.05)',
    },
  },
})

// 导出主题
export const theme = extendTheme({
  colors,
  components: {
    Button,
    Input,
    Card,
  },
  fonts: {
    heading: `-apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif`,
    body: `-apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif`,
  },
  styles: {
    global: {
      html: {
        overflowX: 'clip',
        // overflow-y 需非 visible 才能让 scrollbar-gutter 生效
        overflowY: 'auto',
        // 预留滚动条空间，避免 AlertDialog/Modal 打开时滚动条消失导致页面抖动
        scrollbarGutter: 'stable',
      },
      body: {
        bg: 'myGray.50',
        color: 'myGray.900',
        overflowX: 'clip',
      },
    },
  },
})
