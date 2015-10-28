"""Sphinx Cloudwan theme.

From http://docs.readthedocs.org/en/latest/theme.html.

"""
import os

VERSION = (0, 1, 9)

__version__ = ".".join(str(v) for v in VERSION)
__version_full__ = __version__


def get_html_theme_path():
    """Return list of HTML theme paths."""
    cur_dir = os.path.abspath(os.path.dirname(os.path.dirname(__file__)))
    return cur_dir
