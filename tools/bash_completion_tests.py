#!/usr/bin/env python
import subprocess
import unittest

class Completion():
    def prepare(self, program, command):
        self.program=program
        self.COMP_LINE="%s %s" % (program, command)
        self.COMP_WORDS=self.COMP_LINE.rstrip()
        args=command.split()
        self.COMP_CWORD=len(args)
        self.COMP_POINT=len(self.COMP_LINE)
        if (self.COMP_LINE[-1] == ' '):
            self.COMP_WORDS += " "
            self.COMP_CWORD += 1

    def run(self, completion_file, program, command):
        self.prepare(program, command)
        full_cmdline=r'source {compfile}; COMP_LINE="{COMP_LINE}" COMP_WORDS=({COMP_WORDS}) COMP_CWORD={COMP_CWORD} COMP_POINT={COMP_POINT}; $(complete -p {program} | sed "s/.*-F \\([^ ]*\\) .*/\\1/") && echo ${{COMPREPLY[*]}}'.format(
                compfile=completion_file, COMP_LINE=self.COMP_LINE, COMP_WORDS=self.COMP_WORDS, COMP_POINT=self.COMP_POINT, program=self.program, COMP_CWORD=self.COMP_CWORD
                )
        out = subprocess.Popen(['bash', '-i', '-c', full_cmdline], stdout=subprocess.PIPE)
        return out.communicate()

class CompletionTestCase(unittest.TestCase):

    def assertEqualCompletion(self, program, cline, line, words, cword, point):
        c = Completion()
        c.prepare(program, cline)
        self.assertEqual(c.program, program)
        self.assertEqual(c.COMP_LINE, line)
        self.assertEqual(c.COMP_WORDS, words)
        self.assertEqual(c.COMP_CWORD, cword)
        self.assertEqual(c.COMP_POINT, point)

class BashCompletionTest(unittest.TestCase):
    def run_complete(self, completion_file, program, command, expected):
        stdout,stderr = Completion().run(completion_file, program, command)
        self.assertEqual(stdout.decode("utf-8"), expected + '\n')

class GohanTestCases(BashCompletionTest):
    def test_orig(self):
        self.run_complete("v", "validate v")
        self.run_complete("val", "validate")
        self.run_complete("help","help")
        self.run_complete("--help d","dot")
        self.run_complete("markdown --po","--policy")
        self.run_complete("openapi --t","--template --title")
        self.run_complete("-l","")
        self.run_complete("init-db ","")
        self.run_complete(" ","client validate v init-db idb convert conv server srv test_extensions test_ex migrate mig template run test test openapi markdown dot glace-server gsrv help h")

    def run_complete(self, command, expected):
        completion_file="bash_completion.sh"
        program="gohan"
        super(GohanTestCases, self).run_complete(completion_file, program, command, expected)


if (__name__=='__main__'):
    unittest.main()
