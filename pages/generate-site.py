#!/usr/bin/env python

import cStringIO
import datetime
import os
import subprocess
import re
import sys

CONFIG = {
  'markdown': './Markdown.pl',
  'template': 'template.html',
}

def make_page(tpl, basedir, markdown_filename):
  # we assume markdown_filename ends with '.md'
  html_filename = markdown_filename.rsplit('.', 1)[0] + '.html'
  need_generate = False
  try:
    need_generate = (
      os.stat(os.path.join(basedir, html_filename)).st_mtime
      < os.stat(os.path.join(basedir, markdown_filename)).st_mtime)
  except OSError:
    need_generate = True
  if not need_generate:
    return
  sys.stdout.write('%s -> %s\n' % (markdown_filename, html_filename))

  doc_name = markdown_filename.rsplit('.', 1)[0]
  values = {
    'title': doc_name,
    'basepath': os.path.relpath('.', basedir),
    'update_time': (
        datetime.datetime.fromtimestamp(
            os.stat(os.path.join(basedir, markdown_filename)).st_mtime)
        .strftime('%b %d, %Y')),
  }

  md_doc = cStringIO.StringIO()
  for line in open(os.path.join(basedir, markdown_filename)):
    m = re.match(r'^@(\w+)\s+(.+)$', line.rstrip())
    if m is not None:
      values[m.groups()[0]] = m.groups()[1]
    else:
      md_doc.write(line)

  proc = subprocess.Popen(
      [CONFIG['markdown']], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
  values['content'] = proc.communicate(input=md_doc.getvalue())[0]
  proc.wait()
  f = open(os.path.join(basedir, html_filename), 'w')
  f.write(tpl % values)
  f.close()

def main():
  tpl = open(CONFIG['template']).read()
  for basedir, dirs, files in os.walk('.'):
    for filename in files:
      if filename.endswith('.md'):
        make_page(tpl, basedir, filename)

if __name__ == '__main__':
  main()
